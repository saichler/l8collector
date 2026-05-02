package k8sclient

import (
	"strings"
	"sync"
	"time"
)

// CachedObject is the normalized cache entry served by Exec(job).
type CachedObject struct {
	GVR             string
	Namespace       string
	Name            string
	UID             string
	ResourceVersion string
	Operation       string
	Object          map[string]interface{}
	Related         []map[string]interface{}
	ObservedAt      int64
}

// CollectorCache stores normalized Kubernetes objects for cache-backed reads.
type CollectorCache struct {
	lock    sync.RWMutex
	objects map[string]*CachedObject
}

func NewCollectorCache() *CollectorCache {
	return &CollectorCache{
		objects: make(map[string]*CachedObject),
	}
}

func (c *CollectorCache) Upsert(obj *CachedObject) {
	if obj == nil {
		return
	}
	if obj.ObservedAt == 0 {
		obj.ObservedAt = time.Now().Unix()
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	c.objects[cacheKey(obj.GVR, obj.Namespace, obj.Name)] = obj
}

func (c *CollectorCache) Delete(gvr, namespace, name string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.objects, cacheKey(gvr, namespace, name))
}

func (c *CollectorCache) Get(gvr, namespace, name string) (*CachedObject, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	obj, ok := c.objects[cacheKey(gvr, namespace, name)]
	return obj, ok
}

func (c *CollectorCache) List(gvr, namespace, selector string) []*CachedObject {
	c.lock.RLock()
	defer c.lock.RUnlock()
	totalScanned := 0
	matchedGVR := 0
	matchedNS := 0
	matchedSel := 0
	result := make([]*CachedObject, 0)
	for _, obj := range c.objects {
		if obj == nil {
			continue
		}
		totalScanned++
		if gvr != "" && obj.GVR != gvr {
			continue
		}
		matchedGVR++
		if namespace != "" && obj.Namespace != namespace {
			continue
		}
		matchedNS++
		if !matchesSelector(obj, selector) {
			continue
		}
		matchedSel++
		result = append(result, obj)
	}
	return result
}

func cacheKey(gvr, namespace, name string) string {
	return gvr + "::" + namespace + "::" + name
}

func matchesSelector(obj *CachedObject, selector string) bool {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return true
	}
	labelsValue, ok := nestedValue(obj.Object, []string{"metadata", "labels"})
	if !ok {
		return false
	}
	labels, ok := labelsValue.(map[string]interface{})
	if !ok {
		return false
	}
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tokens := strings.SplitN(part, "=", 2)
		if len(tokens) != 2 {
			return false
		}
		key := strings.TrimSpace(tokens[0])
		expected := strings.TrimSpace(tokens[1])
		actual, ok := labels[key]
		if !ok || stringify(actual) != expected {
			return false
		}
	}
	return true
}
