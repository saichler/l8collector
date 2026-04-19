package k8sclient

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ParseGVR parses resource strings in the form:
// - group/version/resource, e.g. apps/v1/deployments
// - version/resource, e.g. v1/pods
func ParseGVR(raw string) (schema.GroupVersionResource, error) {
	raw = strings.TrimSpace(raw)
	parts := strings.Split(raw, "/")
	switch len(parts) {
	case 2:
		return schema.GroupVersionResource{
			Group:    "",
			Version:  parts[0],
			Resource: parts[1],
		}, nil
	case 3:
		return schema.GroupVersionResource{
			Group:    parts[0],
			Version:  parts[1],
			Resource: parts[2],
		}, nil
	default:
		return schema.GroupVersionResource{}, fmt.Errorf("invalid gvr %q", raw)
	}
}
