package k8sclient

import (
	"context"
	"fmt"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const reaperInterval = 60 * time.Second
const reaperSkipRecentSeconds = 30

// startReaper begins the background reconciliation goroutine exactly once.
func (s *sharedRuntimeState) startReaper() {
	s.mu.Lock()
	if s.reaperStarted {
		s.mu.Unlock()
		return
	}
	s.reaperStarted = true
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-time.After(reaperInterval):
				reapStaleEntries()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func reapStaleEntries() {
	if shared.dynamicClient == nil {
		return
	}

	entries := shared.cache.List("", "", "")
	now := time.Now().Unix()
	checked := 0
	reaped := 0

	for _, entry := range entries {
		if entry.GVR == "" || entry.Name == "" {
			continue
		}
		if now-entry.ObservedAt < reaperSkipRecentSeconds {
			continue
		}

		gvr := parseGVR(entry.GVR)
		if gvr.Resource == "" {
			continue
		}

		var err error
		if entry.Namespace == "" {
			_, err = shared.dynamicClient.Resource(gvr).Get(
				context.TODO(), entry.Name, metav1.GetOptions{})
		} else {
			_, err = shared.dynamicClient.Resource(gvr).Namespace(entry.Namespace).Get(
				context.TODO(), entry.Name, metav1.GetOptions{})
		}
		checked++

		if err != nil && k8serrors.IsNotFound(err) {
			fmt.Printf("[REAPER] stale entry: gvr=%s ns=%s name=%s — removing\n",
				entry.GVR, entry.Namespace, entry.Name)
			shared.cache.Delete(entry.GVR, entry.Namespace, entry.Name)
			if shared.onDelete != nil {
				shared.onDelete(entry.GVR, entry.Namespace, entry.Name)
			}
			reaped++
		}
	}

	if reaped > 0 {
		fmt.Printf("[REAPER] cycle complete: checked=%d reaped=%d\n", checked, reaped)
	}
}

// parseGVR converts the string format "group/version/resource" or
// "version/resource" back to a schema.GroupVersionResource.
func parseGVR(gvrText string) schema.GroupVersionResource {
	parts := strings.Split(gvrText, "/")
	switch len(parts) {
	case 2:
		return schema.GroupVersionResource{
			Group: "", Version: parts[0], Resource: parts[1],
		}
	case 3:
		return schema.GroupVersionResource{
			Group: parts[0], Version: parts[1], Resource: parts[2],
		}
	default:
		if len(parts) >= 3 {
			return schema.GroupVersionResource{
				Group:    strings.Join(parts[:len(parts)-2], "/"),
				Version:  parts[len(parts)-2],
				Resource: parts[len(parts)-1],
			}
		}
		return schema.GroupVersionResource{}
	}
}
