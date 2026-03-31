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
	c.cache = NewCollectorCache()
	c.warmed = make(map[string]bool)
	c.stopCh = make(chan struct{})
	return nil
}

func (c *ClientGoCollector) Protocol() l8tpollaris.L8PProtocol {
	if c.config != nil {
		return c.config.Protocol
	}
	return l8tpollaris.L8PProtocol(0)
}

func (c *ClientGoCollector) Exec(job *l8tpollaris.CJob) {
	if !c.connected {
		if err := c.Connect(); err != nil {
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
	}
	poll, err := pollaris.Poll(job.PollarisName, job.JobName, c.resources)
	if err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	spec, err := ParseCacheSpec(poll.What, poll)
	if err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}
	namespace := resolveSpecValue(spec.Namespace, spec.NamespaceFromArg, job.Arguments)
	name := resolveSpecValue(spec.Name, spec.NameFromArg, job.Arguments)
	selector := resolveSpecValue(spec.Selector, spec.SelectorFromArg, job.Arguments)
	err = c.ensureWarm(spec, namespace)
	if err != nil {
		job.Error = err.Error()
		job.ErrorCount++
		return
	}

	switch spec.Result {
	case ResultMap:
		item, ok := c.cache.Get(spec.GVR, namespace, name)
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
		err = enc.Add(cmap)
		if err != nil {
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		job.Result = enc.Data()
	case ResultTable:
		items := c.cache.List(spec.GVR, namespace, selector)
		tbl, err := BuildCTable(items, spec.Fields, spec.ColumnNames)
		if err != nil {
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		enc := object.NewEncode()
		err = enc.Add(tbl)
		if err != nil {
			job.Error = err.Error()
			job.ErrorCount++
			return
		}
		job.Result = enc.Data()
	default:
		job.Error = "unsupported cache result type " + spec.Result
		job.ErrorCount++
		return
	}
	job.Error = ""
	job.ErrorCount = 0
}

func (c *ClientGoCollector) Connect() error {
	if c.cache == nil {
		c.cache = NewCollectorCache()
	}
	if c.warmed == nil {
		c.warmed = make(map[string]bool)
	}
	if c.stopCh == nil {
		c.stopCh = make(chan struct{})
	}
	cfg, err := c.kubeConfig()
	if err != nil {
		return err
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return err
	}
	c.restConfig = cfg
	c.dynamicClient = client
	c.connected = true
	return nil
}

func (c *ClientGoCollector) Disconnect() error {
	c.connected = false
	if c.stopCh != nil {
		close(c.stopCh)
		c.stopCh = nil
	}
	c.resources = nil
	c.config = nil
	c.cache = nil
	c.restConfig = nil
	c.dynamicClient = nil
	c.warmed = nil
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

func (c *ClientGoCollector) ensureWarm(spec *CacheSpec, namespace string) error {
	if spec == nil {
		return nil
	}
	return c.ensureWatching(spec.GVR, namespace)
}

func (c *ClientGoCollector) startInformer(gvr schema.GroupVersionResource, gvrText, namespace, warmKey string) error {
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(c.dynamicClient, 0, namespace, nil)
	informer := factory.ForResource(gvr).Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			c.Upsert(normalizeObject(gvrText, item, "ADD"))
		},
		UpdateFunc: func(_, obj interface{}) {
			item, ok := obj.(*unstructured.Unstructured)
			if !ok {
				return
			}
			c.Upsert(normalizeObject(gvrText, item, "UPDATE"))
		},
		DeleteFunc: func(obj interface{}) {
			item, ok := extractDeletedObject(obj)
			if !ok {
				return
			}
			c.Delete(gvrText, item.GetNamespace(), item.GetName())
		},
	})
	factory.Start(c.stopCh)
	if !cache.WaitForCacheSync(c.stopCh, informer.HasSynced) {
		return fmt.Errorf("failed to sync informer cache for %s", warmKey)
	}
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
		c.warmMu.Unlock()
		return nil
	}
	c.warmMu.Unlock()

	gvr, err := ParseGVR(gvrText)
	if err != nil {
		return err
	}
	err = c.startInformer(gvr, gvrText, namespace, warmKey)
	if err != nil {
		return err
	}
	c.warmMu.Lock()
	c.warmed[warmKey] = true
	c.warmMu.Unlock()
	return nil
}
