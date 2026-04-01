package k8sclient

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientGoCollector is a cache-first Kubernetes collector.
//
// All instances share a single connection, cache, and set of informers via
// the package-level shared runtime. Instance fields hold convenience
// references obtained from the shared runtime at Init/Connect time.
type ClientGoCollector struct {
	resources ifs.IResources
	config    *l8tpollaris.L8PHostProtocol
}

func (c *ClientGoCollector) Init(config *l8tpollaris.L8PHostProtocol, resources ifs.IResources) error {
	c.resources = resources
	c.config = config
	shared.init()
	return c.ensureAdmissionServerStarted()
}

func (c *ClientGoCollector) Protocol() l8tpollaris.L8PProtocol {
	if c.config != nil {
		return c.config.Protocol
	}
	return l8tpollaris.L8PProtocol(0)
}

func (c *ClientGoCollector) Exec(job *l8tpollaris.CJob) {
	c.log(ifs.Debug_Level, "Exec start pollaris=%s job=%s target=%s", job.PollarisName, job.JobName, job.TargetId)

	if !shared.connected {
		if err := c.Connect(); err != nil {
			c.log(ifs.Debug_Level, "Exec connect error: %s", err.Error())
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
	}

	poll, err := pollaris.Poll(job.PollarisName, job.JobName, c.resources)
	if err != nil {
		c.log(ifs.Debug_Level, "Exec poll lookup error: %s", err.Error())
		job.Error = err.Error()
		job.ErrorCount++
		return
	}

	spec, err := ParseCacheSpec(poll.What, poll)
	if err != nil {
		c.log(ifs.Debug_Level, "Exec parse spec error: %s", err.Error())
		job.Error = err.Error()
		job.ErrorCount++
		return
	}

	namespace := resolveSpecValue(spec.Namespace, spec.NamespaceFromArg, job.Arguments)
	name := resolveSpecValue(spec.Name, spec.NameFromArg, job.Arguments)

	if err = c.ensureWatching(spec.GVR, namespace); err != nil {
		c.log(ifs.Debug_Level, "Exec warm error: %s", err.Error())
		job.Error = err.Error()
		job.ErrorCount++
		return
	}

	switch spec.Result {
	case ResultMap:
		c.execMap(job, spec, namespace, name)
	case ResultTable:
		selector := resolveSpecValue(spec.Selector, spec.SelectorFromArg, job.Arguments)
		c.execTable(job, spec, namespace, selector)
	default:
		job.Error = "unsupported cache result type " + spec.Result
		job.ErrorCount++
		return
	}

	if job.Error != "" {
		return
	}
	job.ErrorCount = 0
	c.log(ifs.Debug_Level, "Exec done job=%s result-bytes=%d", job.JobName, len(job.Result))
}

func (c *ClientGoCollector) execMap(job *l8tpollaris.CJob, spec *CacheSpec, namespace, name string) {
	item, ok := shared.cache.Get(spec.GVR, namespace, name)
	if !ok {
		job.Error = fmt.Sprintf("cache miss for %s/%s/%s", spec.GVR, namespace, name)
		job.ErrorCount++
		return
	}
	cmap, err := BuildCMap(item, spec.Fields)
	if err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	enc := object.NewEncode()
	if err = enc.Add(cmap); err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	job.Result = enc.Data()
}

func (c *ClientGoCollector) execTable(job *l8tpollaris.CJob, spec *CacheSpec, namespace, selector string) {
	items := shared.cache.List(spec.GVR, namespace, selector)
	tbl, err := BuildCTable(items, spec.Fields, spec.ColumnNames)
	if err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	enc := object.NewEncode()
	if err = enc.Add(tbl); err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	job.Result = enc.Data()
}

func (c *ClientGoCollector) Connect() error {
	shared.init()
	_, _, err := shared.connect(c.kubeConfig)
	return err
}

func (c *ClientGoCollector) Disconnect() error {
	shared.disconnect(c.logger())
	c.resources = nil
	c.config = nil
	return nil
}

func (c *ClientGoCollector) Online() bool {
	return shared.connected
}

// AdmissionHandler returns an HTTP handler that processes Kubernetes
// admission review requests. The handler always allows the request —
// it is used for observation/cache-population only, not for blocking.
func (c *ClientGoCollector) AdmissionHandler() (http.Handler, error) {
	rules, err := WebhookRulesFromBootModels()
	if err != nil {
		return nil, err
	}
	return NewAdmissionHandler(rules, c.handleAdmissionEvent), nil
}

func (c *ClientGoCollector) ensureAdmissionServerStarted() error {
	if c == nil || c.config == nil || c.config.Protocol != l8tpollaris.L8PProtocol_L8PKubernetesAPI {
		return nil
	}
	if os.Getenv("ClusterName") == "" {
		return nil
	}
	return shared.ensureAdmissionServer(func() error {
		if err := c.Connect(); err != nil {
			return err
		}
		return c.StartAdmissionServer(c.resources)
	})
}

func (c *ClientGoCollector) handleAdmissionEvent(event AdmissionEvent) error {
	gvrText := event.Version + "/" + event.Resource
	if event.Group != "" {
		gvrText = event.Group + "/" + gvrText
	}
	if err := c.ensureWatching(gvrText, event.Namespace); err != nil {
		return err
	}
	// The informer started by ensureWatching will observe the same
	// mutation via the API server watch stream. We update the cache here
	// as well for lower latency on the first event before the informer
	// catches up.
	if event.Operation == "DELETE" {
		shared.cache.Delete(gvrText, event.Namespace, event.Name)
		return nil
	}
	if event.Object != nil {
		shared.cache.Upsert(normalizeObject(gvrText, event.Object, event.Operation))
	}
	return nil
}

// ensureWatching starts an informer for the given GVR+namespace exactly
// once, using sync.Once per warm key to prevent duplicate informers.
//
// If an all-namespace informer is already running for this GVR, a
// namespace-specific request is a no-op (the all-namespace informer
// already covers it). This prevents duplicate watches.
func (c *ClientGoCollector) ensureWatching(gvrText, namespace string) error {
	if shared.dynamicClient == nil || gvrText == "" {
		return nil
	}

	// If an all-namespace informer already covers this GVR, skip.
	allNsKey := cacheKey(gvrText, "", "*")
	if namespace != "" && shared.isWarmed(allNsKey) {
		return nil
	}

	warmKey := cacheKey(gvrText, namespace, "*")
	if shared.isWarmed(warmKey) {
		return nil
	}

	var startErr error
	once := shared.onceForKey(warmKey)
	once.Do(func() {
		gvr, err := ParseGVR(gvrText)
		if err != nil {
			startErr = err
			return
		}
		startErr = c.startInformer(gvr, gvrText, namespace, warmKey)
		if startErr == nil {
			shared.markWarmed(warmKey)
		}
	})
	return startErr
}

func (c *ClientGoCollector) startInformer(gvr schema.GroupVersionResource, gvrText, namespace, warmKey string) error {
	c.log(ifs.Debug_Level, "startInformer gvr=%s namespace=%s", gvrText, namespace)
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(shared.dynamicClient, 0, namespace, nil)
	informer := factory.ForResource(gvr).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			shared.cache.Upsert(normalizeObject(gvrText, item, "ADD"))
		},
		UpdateFunc: func(_, obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			shared.cache.Upsert(normalizeObject(gvrText, item, "UPDATE"))
		},
		DeleteFunc: func(obj interface{}) {
			item, ok := extractDeletedObject(obj)
			if !ok {
				return
			}
			shared.cache.Delete(gvrText, item.GetNamespace(), item.GetName())
		},
	})
	factory.Start(shared.stopCh)
	if !cache.WaitForCacheSync(shared.stopCh, informer.HasSynced) {
		return fmt.Errorf("failed to sync informer cache for %s", warmKey)
	}
	c.log(ifs.Debug_Level, "startInformer synced warmKey=%s", warmKey)
	return nil
}

func (c *ClientGoCollector) kubeConfig() (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}
	if c.resources != nil && c.config != nil && c.config.CredId != "" {
		_, _, kubeconfig, _, credErr := c.resources.Security().Credential(c.config.CredId, "kubeconfig", c.resources)
		if credErr == nil {
			cfg, cfgErr := restConfigFromString(kubeconfig)
			if cfgErr == nil {
				return cfg, nil
			}
			return nil, cfgErr
		}
	}
	envPath := os.Getenv("KUBECONFIG")
	if envPath != "" {
		cfg, cfgErr := clientcmd.BuildConfigFromFlags("", envPath)
		if cfgErr == nil {
			return cfg, nil
		}
	}
	for _, candidate := range []string{"admin.conf", filepath.Join("go", "admin.conf")} {
		if _, statErr := os.Stat(candidate); statErr != nil {
			continue
		}
		cfg, cfgErr := clientcmd.BuildConfigFromFlags("", candidate)
		if cfgErr == nil {
			return cfg, nil
		}
	}
	return nil, err
}

func restConfigFromString(kubeconfig string) (*rest.Config, error) {
	data, decodeErr := base64.StdEncoding.DecodeString(kubeconfig)
	if decodeErr != nil {
		if strings.Contains(kubeconfig, "apiVersion:") {
			data = []byte(kubeconfig)
		} else {
			return nil, decodeErr
		}
	}
	return clientcmd.RESTConfigFromKubeConfig(data)
}

func normalizeObject(gvr string, item *unstructured.Unstructured, operation string) *CachedObject {
	if item == nil {
		return nil
	}
	obj := item.DeepCopy().Object
	enrichObject(gvr, obj)
	return &CachedObject{
		GVR:             gvr,
		Namespace:       item.GetNamespace(),
		Name:            item.GetName(),
		UID:             string(item.GetUID()),
		ResourceVersion: item.GetResourceVersion(),
		Operation:       operation,
		Object:          obj,
		Related:         make([]map[string]interface{}, 0),
	}
}

func extractDeletedObject(obj interface{}) (*unstructured.Unstructured, bool) {
	item, ok := obj.(*unstructured.Unstructured)
	if ok {
		return item, true
	}
	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		return nil, false
	}
	item, ok = tombstone.Obj.(*unstructured.Unstructured)
	return item, ok
}

// log writes through the collector's resources logger if available,
// otherwise falls back to fmt.Println.
func (c *ClientGoCollector) log(level ifs.LogLevel, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	logger := c.logger()
	if logger == nil {
		fmt.Println("[k8sclient] " + msg)
		return
	}
	switch level {
	case ifs.Error_Level:
		logger.Error(msg)
	case ifs.Warning_Level:
		logger.Warning(msg)
	case ifs.Info_Level:
		logger.Info(msg)
	default:
		logger.Debug(msg)
	}
}

func (c *ClientGoCollector) logger() ifs.ILogger {
	if c.resources != nil {
		return c.resources.Logger()
	}
	return nil
}
