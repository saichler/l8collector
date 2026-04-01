package k8sclient

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/saichler/l8pollaris/go/pollaris"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	"github.com/saichler/l8types/go/ifs"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var sharedRuntime struct {
	lock           sync.Mutex
	cache          *CollectorCache
	restConfig     *rest.Config
	dynamicClient  dynamic.Interface
	warmed         map[string]bool
	stopCh         chan struct{}
	connected      bool
	serverStarted  bool
	serverStarting bool
}

// ClientGoCollector is a cache-first Kubernetes collector.
//
// The current repo-local implementation focuses on the cache-backed Exec(job)
// path. Admission/informer cache population will be added on top of this type
// once the external protocol and client-go dependency updates are vendored.
type ClientGoCollector struct {
	resources     ifs.IResources
	config        *l8tpollaris.L8PHostProtocol
	cache         *CollectorCache
	restConfig    *rest.Config
	dynamicClient dynamic.Interface
	connected     bool
	warmMu        sync.Mutex
	warmed        map[string]bool
	stopCh        chan struct{}
}

func (c *ClientGoCollector) Init(config *l8tpollaris.L8PHostProtocol, resources ifs.IResources) error {
	c.resources = resources
	c.config = config
	c.initSharedRuntimeState()
	if err := c.ensureAdmissionServerStarted(); err != nil {
		return err
	}
	return nil
}

func (c *ClientGoCollector) initSharedRuntimeState() {
	sharedRuntime.lock.Lock()
	defer sharedRuntime.lock.Unlock()
	if sharedRuntime.cache == nil {
		sharedRuntime.cache = NewCollectorCache()
	}
	if sharedRuntime.warmed == nil {
		sharedRuntime.warmed = make(map[string]bool)
	}
	if sharedRuntime.stopCh == nil {
		sharedRuntime.stopCh = make(chan struct{})
	}
	c.cache = sharedRuntime.cache
	c.warmed = sharedRuntime.warmed
	c.stopCh = sharedRuntime.stopCh
	c.restConfig = sharedRuntime.restConfig
	c.dynamicClient = sharedRuntime.dynamicClient
	c.connected = sharedRuntime.connected
}

func (c *ClientGoCollector) Protocol() l8tpollaris.L8PProtocol {
	if c.config != nil {
		return c.config.Protocol
	}
	return l8tpollaris.L8PProtocol(0)
}

func (c *ClientGoCollector) Exec(job *l8tpollaris.CJob) {
	fmt.Println("[adcon-debug] k8sclient.Exec start pollaris=", job.PollarisName, "job=", job.JobName, "target=", job.TargetId)
	if !c.connected {
		if err := c.Connect(); err != nil {
			fmt.Println("[adcon-debug] k8sclient.Exec connect-error=", err.Error())
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, c.resources)
	if err != nil {
		fmt.Println("[adcon-debug] k8sclient.Exec poll-lookup-error=", err.Error())
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	spec, err := ParseCacheSpec(poll.What, poll)
	if err != nil {
		fmt.Println("[adcon-debug] k8sclient.Exec parse-spec-error=", err.Error())
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	namespace := resolveSpecValue(spec.Namespace, spec.NamespaceFromArg, job.Arguments)
	name := resolveSpecValue(spec.Name, spec.NameFromArg, job.Arguments)
	selector := resolveSpecValue(spec.Selector, spec.SelectorFromArg, job.Arguments)
	fmt.Println("[adcon-debug] k8sclient.Exec spec gvr=", spec.GVR, "result=", spec.Result, "mode=", spec.Mode, "namespace=", namespace, "name=", name, "selector=", selector)
	err = c.ensureWarm(spec, namespace)
	if err != nil {
		fmt.Println("[adcon-debug] k8sclient.Exec warm-error=", err.Error())
		job.Error = err.Error()
		job.ErrorCount++
		return
	}

	switch spec.Result {
	case ResultMap:
		item, ok := c.cache.Get(spec.GVR, namespace, name)
		if !ok {
			fmt.Println("[adcon-debug] k8sclient.Exec cache-miss gvr=", spec.GVR, "namespace=", namespace, "name=", name)
			job.Error = fmt.Sprintf("cache miss for %s/%s/%s", spec.GVR, namespace, name)
			job.ErrorCount++
			return
		}
		cmap, err := BuildCMap(item, spec.Fields)
		if err != nil {
			fmt.Println("[adcon-debug] k8sclient.Exec build-cmap-error=", err.Error())
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		enc := object.NewEncode()
		err = enc.Add(cmap)
		if err != nil {
			fmt.Println("[adcon-debug] k8sclient.Exec encode-cmap-error=", err.Error())
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		job.Result = enc.Data()
		fmt.Println("[adcon-debug] k8sclient.Exec cmap-result-bytes=", len(job.Result))
	case ResultTable:
		items := c.cache.List(spec.GVR, namespace, selector)
		fmt.Println("[adcon-debug] k8sclient.Exec table-items=", len(items))
		tbl, err := BuildCTable(items, spec.Fields, spec.ColumnNames)
		if err != nil {
			fmt.Println("[adcon-debug] k8sclient.Exec build-ctable-error=", err.Error())
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		enc := object.NewEncode()
		err = enc.Add(tbl)
		if err != nil {
			fmt.Println("[adcon-debug] k8sclient.Exec encode-ctable-error=", err.Error())
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		job.Result = enc.Data()
		fmt.Println("[adcon-debug] k8sclient.Exec ctable-result-bytes=", len(job.Result), "rows=", len(tbl.Rows))
	default:
		fmt.Println("[adcon-debug] k8sclient.Exec unsupported-result=", spec.Result)
		job.Error = "unsupported cache result type " + spec.Result
		job.ErrorCount++
		return
	}
	job.Error = ""
	job.ErrorCount = 0
	fmt.Println("[adcon-debug] k8sclient.Exec done job=", job.JobName, "result-bytes=", len(job.Result))
}

func (c *ClientGoCollector) Connect() error {
	sharedRuntime.lock.Lock()
	if sharedRuntime.cache == nil {
		sharedRuntime.cache = NewCollectorCache()
	}
	if sharedRuntime.warmed == nil {
		sharedRuntime.warmed = make(map[string]bool)
	}
	if sharedRuntime.stopCh == nil {
		sharedRuntime.stopCh = make(chan struct{})
	}
	if sharedRuntime.connected && sharedRuntime.dynamicClient != nil {
		c.cache = sharedRuntime.cache
		c.warmed = sharedRuntime.warmed
		c.stopCh = sharedRuntime.stopCh
		c.restConfig = sharedRuntime.restConfig
		c.dynamicClient = sharedRuntime.dynamicClient
		c.connected = true
		sharedRuntime.lock.Unlock()
		return nil
	}
	sharedRuntime.lock.Unlock()

	cfg, err := c.kubeConfig()
	if err != nil {
		return err
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}

	sharedRuntime.lock.Lock()
	defer sharedRuntime.lock.Unlock()
	if sharedRuntime.cache == nil {
		sharedRuntime.cache = NewCollectorCache()
	}
	if sharedRuntime.warmed == nil {
		sharedRuntime.warmed = make(map[string]bool)
	}
	if sharedRuntime.stopCh == nil {
		sharedRuntime.stopCh = make(chan struct{})
	}
	if sharedRuntime.connected && sharedRuntime.dynamicClient != nil {
		c.cache = sharedRuntime.cache
		c.warmed = sharedRuntime.warmed
		c.stopCh = sharedRuntime.stopCh
		c.restConfig = sharedRuntime.restConfig
		c.dynamicClient = sharedRuntime.dynamicClient
		c.connected = true
		return nil
	}
	sharedRuntime.restConfig = cfg
	sharedRuntime.dynamicClient = client
	sharedRuntime.connected = true
	c.cache = sharedRuntime.cache
	c.warmed = sharedRuntime.warmed
	c.stopCh = sharedRuntime.stopCh
	c.restConfig = cfg
	c.dynamicClient = client
	c.connected = true
	return nil
}

func (c *ClientGoCollector) Disconnect() error {
	c.connected = false
	c.resources = nil
	c.config = nil
	return nil
}

func (c *ClientGoCollector) Online() bool {
	return c.connected
}

// Upsert inserts or replaces a normalized object in the collector cache.
func (c *ClientGoCollector) Upsert(obj *CachedObject) {
	if c.cache == nil {
		c.cache = NewCollectorCache()
	}
	c.cache.Upsert(obj)
}

// Delete removes a normalized object from the collector cache.
func (c *ClientGoCollector) Delete(gvr, namespace, name string) {
	if c.cache == nil {
		return
	}
	c.cache.Delete(gvr, namespace, name)
}

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

	sharedRuntime.lock.Lock()
	if sharedRuntime.serverStarted || sharedRuntime.serverStarting {
		sharedRuntime.lock.Unlock()
		return nil
	}
	sharedRuntime.serverStarting = true
	sharedRuntime.lock.Unlock()

	err := c.Connect()
	if err == nil {
		err = c.StartAdmissionServer(c.resources)
	}

	sharedRuntime.lock.Lock()
	defer sharedRuntime.lock.Unlock()
	sharedRuntime.serverStarting = false
	if err == nil {
		sharedRuntime.serverStarted = true
	}
	return err
}

func (c *ClientGoCollector) WarmUpFromBootModels() error {
	if !c.connected {
		if err := c.Connect(); err != nil {
			return err
		}
	}
	specs, err := CacheSpecsFromBootModels()
	if err != nil {
		return err
	}
	for _, spec := range specs {
		if spec == nil {
			continue
		}
		if err = c.ensureWarm(spec, spec.Namespace); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClientGoCollector) ensureWarm(spec *CacheSpec, namespace string) error {
	if spec == nil {
		return nil
	}
	return c.ensureWatching(spec.GVR, namespace)
}

func (c *ClientGoCollector) startInformer(gvr schema.GroupVersionResource, gvrText, namespace, warmKey string) error {
	fmt.Println("[adcon-debug] k8sclient.startInformer gvr=", gvrText, "namespace=", namespace, "warmKey=", warmKey)
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(c.dynamicClient, 0, namespace, nil)
	informer := factory.ForResource(gvr).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			fmt.Println("[adcon-debug] k8sclient.informer add gvr=", gvrText, "ns=", item.GetNamespace(), "name=", item.GetName())
			c.Upsert(normalizeObject(gvrText, item, "ADD"))
		},
		UpdateFunc: func(_, obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			fmt.Println("[adcon-debug] k8sclient.informer update gvr=", gvrText, "ns=", item.GetNamespace(), "name=", item.GetName())
			c.Upsert(normalizeObject(gvrText, item, "UPDATE"))
		},
		DeleteFunc: func(obj interface{}) {
			item, ok := extractDeletedObject(obj)
			if !ok {
				return
			}
			fmt.Println("[adcon-debug] k8sclient.informer delete gvr=", gvrText, "ns=", item.GetNamespace(), "name=", item.GetName())
			c.Delete(gvrText, item.GetNamespace(), item.GetName())
		},
	})
	factory.Start(c.stopCh)
	if !cache.WaitForCacheSync(c.stopCh, informer.HasSynced) {
		fmt.Println("[adcon-debug] k8sclient.startInformer sync-failed warmKey=", warmKey)
		return fmt.Errorf("failed to sync informer cache for %s", warmKey)
	}
	fmt.Println("[adcon-debug] k8sclient.startInformer synced warmKey=", warmKey)
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
	obj := item.Object
	related := make([]map[string]interface{}, 0)
	return &CachedObject{
		GVR:             gvr,
		Namespace:       item.GetNamespace(),
		Name:            item.GetName(),
		UID:             string(item.GetUID()),
		ResourceVersion: item.GetResourceVersion(),
		Operation:       operation,
		Object:          obj,
		Related:         related,
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

func (c *ClientGoCollector) handleAdmissionEvent(event AdmissionEvent) error {
	gvrText := event.Version + "/" + event.Resource
	if event.Group != "" {
		gvrText = event.Group + "/" + gvrText
	}
	if err := c.ensureWatching(gvrText, event.Namespace); err != nil {
		return err
	}
	if event.Operation == "DELETE" {
		c.Delete(gvrText, event.Namespace, event.Name)
		return nil
	}
	if event.Object != nil {
		c.Upsert(normalizeObject(gvrText, event.Object, event.Operation))
	}
	return nil
}

func (c *ClientGoCollector) ensureWatching(gvrText, namespace string) error {
	if c.dynamicClient == nil || gvrText == "" {
		return nil
	}
	warmKey := cacheKey(gvrText, namespace, "*")
	c.warmMu.Lock()
	if c.warmed[warmKey] {
		fmt.Println("[adcon-debug] k8sclient.ensureWatching already-warm=", warmKey)
		c.warmMu.Unlock()
		return nil
	}
	c.warmMu.Unlock()
	fmt.Println("[adcon-debug] k8sclient.ensureWatching warming=", warmKey)

	gvr, err := ParseGVR(gvrText)
	if err != nil {
		fmt.Println("[adcon-debug] k8sclient.ensureWatching parse-gvr-error=", err.Error())
		return err
	}
	err = c.startInformer(gvr, gvrText, namespace, warmKey)
	if err != nil {
		fmt.Println("[adcon-debug] k8sclient.ensureWatching start-informer-error=", err.Error())
		return err
	}
	c.warmMu.Lock()
	c.warmed[warmKey] = true
	c.warmMu.Unlock()
	fmt.Println("[adcon-debug] k8sclient.ensureWatching warmed=", warmKey)
	return nil
}
