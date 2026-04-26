package k8sclient

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// enrichObject adds kubectl-equivalent computed fields to the raw K8s
// object map. These are stored under the "_k" key so they don't collide
// with native K8s fields and can be referenced in poll specs as "_k.<name>".
//
// The goal is to produce the same attribute values that "kubectl get -o wide"
// displays so that the k8sclient protocol is format-compatible with the
// kubectl protocol when processed by the downstream parser.
func enrichObject(gvr string, obj map[string]interface{}) {
	if obj == nil {
		return
	}
	resource := resourceFromGVR(gvr)
	computed := make(map[string]interface{})

	switch resource {
	case "nodes":
		enrichNode(obj, computed)
	case "pods":
		enrichPod(obj, computed)
	case "deployments":
		enrichDeployment(obj, computed)
	case "statefulsets":
		enrichStatefulSet(obj, computed)
	case "daemonsets":
		enrichDaemonSet(obj, computed)
	case "services":
		enrichService(obj, computed)
	}

	// Always compute relative age from creationTimestamp.
	enrichAge(obj, computed)

	if len(computed) > 0 {
		obj["_k"] = computed
	}
}

func resourceFromGVR(gvr string) string {
	parts := strings.Split(gvr, "/")
	return parts[len(parts)-1]
}

// --- node ---

func enrichNode(obj map[string]interface{}, out map[string]interface{}) {
	out["roles"] = nodeRoles(obj)
	out["internalip"] = nodeAddress(obj, "InternalIP")
	out["externalip"] = nodeAddress(obj, "ExternalIP")
	out["status"] = nodeReadyStatus(obj)
}

// nodeReadyStatus mirrors `kubectl get nodes`' STATUS column. It reads
// status.conditions[type=Ready].status and returns "Ready" / "NotReady".
// "Unknown" is returned only when the conditions array is absent or the
// Ready condition cannot be located — never as a silent fallback for an
// unrecognized value.
func nodeReadyStatus(obj map[string]interface{}) string {
	conditions, ok := nestedSlice(obj, "status", "conditions")
	if !ok || len(conditions) == 0 {
		return "Unknown"
	}
	for _, entry := range conditions {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if fmt.Sprint(m["type"]) != "Ready" {
			continue
		}
		if fmt.Sprint(m["status"]) == "True" {
			return "Ready"
		}
		return "NotReady"
	}
	return "Unknown"
}

func nodeRoles(obj map[string]interface{}) string {
	labels, _ := nestedMap(obj, "metadata", "labels")
	if labels == nil {
		return "<none>"
	}
	roles := make([]string, 0)
	prefix := "node-role.kubernetes.io/"
	for key := range labels {
		if strings.HasPrefix(key, prefix) {
			roles = append(roles, key[len(prefix):])
		}
	}
	if len(roles) == 0 {
		return "<none>"
	}
	sort.Strings(roles)
	return strings.Join(roles, ",")
}

func nodeAddress(obj map[string]interface{}, addrType string) string {
	addresses, _ := nestedSlice(obj, "status", "addresses")
	for _, entry := range addresses {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if fmt.Sprint(m["type"]) == addrType {
			return fmt.Sprint(m["address"])
		}
	}
	return "<none>"
}

// --- pod ---

func enrichPod(obj map[string]interface{}, out map[string]interface{}) {
	statuses, _ := nestedSlice(obj, "status", "containerStatuses")
	ready, total, restarts := 0, len(statuses), 0
	for _, entry := range statuses {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if b, ok := m["ready"].(bool); ok && b {
			ready++
		}
		if r, ok := toInt(m["restartCount"]); ok {
			restarts += r
		}
	}
	out["ready"] = fmt.Sprintf("%d/%d", ready, total)
	out["restarts"] = fmt.Sprintf("%d", restarts)
	nominatedNode, ok := nestedValue(obj, []string{"status", "nominatedNodeName"})
	if ok && nominatedNode != nil && fmt.Sprint(nominatedNode) != "" {
		out["nominatednode"] = fmt.Sprint(nominatedNode)
	} else {
		out["nominatednode"] = "<none>"
	}
}

// --- deployment ---

func enrichDeployment(obj map[string]interface{}, out map[string]interface{}) {
	replicas := intOr(obj, 0, "spec", "replicas")
	readyReplicas := intOr(obj, 0, "status", "readyReplicas")
	out["ready"] = fmt.Sprintf("%d/%d", readyReplicas, replicas)
	// Pre-stringify updatedReplicas / availableReplicas. K8s omits these
	// JSON fields when they are zero, which would leave the proto string
	// fields empty; intOr(0) substitutes a real "0". The enrichment also
	// sidesteps the int64 → string assignment path in the parser, which
	// previously emitted rune characters or "[]" for these counts.
	out["uptodate"] = fmt.Sprint(intOr(obj, 0, "status", "updatedReplicas"))
	out["available"] = fmt.Sprint(intOr(obj, 0, "status", "availableReplicas"))
	enrichPodTemplate(obj, out)
	out["selector"] = matchLabelsString(obj, "spec", "selector", "matchLabels")
}

// --- statefulset ---

func enrichStatefulSet(obj map[string]interface{}, out map[string]interface{}) {
	replicas := intOr(obj, 0, "spec", "replicas")
	readyReplicas := intOr(obj, 0, "status", "readyReplicas")
	out["ready"] = fmt.Sprintf("%d/%d", readyReplicas, replicas)
	enrichPodTemplate(obj, out)
}

// --- daemonset ---

func enrichDaemonSet(obj map[string]interface{}, out map[string]interface{}) {
	// Same int → string normalization rationale as enrichDeployment: K8s
	// returns these as int64, but the proto fields are `string`, and the
	// parser's int64 → string fallback emits rune garbage or "[]" when the
	// JSON key is missing. Pre-stringifying here gives the parser plain
	// strings end-to-end.
	out["desired"] = fmt.Sprint(intOr(obj, 0, "status", "desiredNumberScheduled"))
	out["current"] = fmt.Sprint(intOr(obj, 0, "status", "currentNumberScheduled"))
	out["ready"] = fmt.Sprint(intOr(obj, 0, "status", "numberReady"))
	out["uptodate"] = fmt.Sprint(intOr(obj, 0, "status", "updatedNumberScheduled"))
	out["available"] = fmt.Sprint(intOr(obj, 0, "status", "numberAvailable"))
	out["nodeselector"] = nodeSelectorString(obj)
	enrichPodTemplate(obj, out)
	out["selector"] = matchLabelsString(obj, "spec", "selector", "matchLabels")
}

// --- service ---

func enrichService(obj map[string]interface{}, out map[string]interface{}) {
	out["externalip"] = serviceExternalIP(obj)
	out["ports"] = servicePortsString(obj)
}

func serviceExternalIP(obj map[string]interface{}) string {
	if ips, ok := nestedSlice(obj, "spec", "externalIPs"); ok && len(ips) > 0 {
		parts := make([]string, 0, len(ips))
		for _, ip := range ips {
			parts = append(parts, fmt.Sprint(ip))
		}
		return strings.Join(parts, ",")
	}
	if ingress, ok := nestedSlice(obj, "status", "loadBalancer", "ingress"); ok && len(ingress) > 0 {
		parts := make([]string, 0, len(ingress))
		for _, entry := range ingress {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			if ip, ok := m["ip"]; ok {
				parts = append(parts, fmt.Sprint(ip))
			} else if hostname, ok := m["hostname"]; ok {
				parts = append(parts, fmt.Sprint(hostname))
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, ",")
		}
	}
	return "<none>"
}

func servicePortsString(obj map[string]interface{}) string {
	ports, _ := nestedSlice(obj, "spec", "ports")
	if len(ports) == 0 {
		return "<none>"
	}
	parts := make([]string, 0, len(ports))
	for _, entry := range ports {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		port := fmt.Sprint(m["port"])
		protocol := "TCP"
		if p, ok := m["protocol"]; ok {
			protocol = fmt.Sprint(p)
		}
		s := port + "/" + protocol
		if np, ok := m["nodePort"]; ok {
			npInt, isNum := toInt(np)
			if isNum && npInt > 0 {
				s = port + ":" + fmt.Sprint(npInt) + "/" + protocol
			}
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, ",")
}

// --- shared helpers for pod template ---

func enrichPodTemplate(obj map[string]interface{}, out map[string]interface{}) {
	containers, _ := nestedSlice(obj, "spec", "template", "spec", "containers")
	names := make([]string, 0, len(containers))
	images := make([]string, 0, len(containers))
	for _, entry := range containers {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if n, ok := m["name"]; ok {
			names = append(names, fmt.Sprint(n))
		}
		if img, ok := m["image"]; ok {
			images = append(images, fmt.Sprint(img))
		}
	}
	out["containers"] = strings.Join(names, ",")
	out["images"] = strings.Join(images, ",")
}

func nodeSelectorString(obj map[string]interface{}) string {
	ns, _ := nestedMap(obj, "spec", "template", "spec", "nodeSelector")
	if len(ns) == 0 {
		return "<none>"
	}
	return mapToSelectorString(ns)
}

func matchLabelsString(obj map[string]interface{}, path ...string) string {
	m, _ := nestedMap(obj, path...)
	if len(m) == 0 {
		return "<none>"
	}
	return mapToSelectorString(m)
}

func mapToSelectorString(m map[string]interface{}) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+fmt.Sprint(m[k]))
	}
	return strings.Join(parts, ",")
}

// --- age ---

func enrichAge(obj map[string]interface{}, out map[string]interface{}) {
	tsRaw, ok := nestedValue(obj, []string{"metadata", "creationTimestamp"})
	if !ok {
		return
	}
	tsStr, ok := tsRaw.(string)
	if !ok {
		return
	}
	t, err := time.Parse(time.RFC3339, tsStr)
	if err != nil {
		return
	}
	out["age"] = formatRelativeAge(time.Since(t))
}

func formatRelativeAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	seconds := int(d.Seconds())
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := seconds / 60
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	remainMinutes := minutes % 60
	if hours < 24 {
		if remainMinutes > 0 {
			return fmt.Sprintf("%dh%dm", hours, remainMinutes)
		}
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	remainHours := hours % 24
	if days < 365 {
		if remainHours > 0 {
			return fmt.Sprintf("%dd%dh", days, remainHours)
		}
		return fmt.Sprintf("%dd", days)
	}
	years := days / 365
	remainDays := days % 365
	if remainDays > 0 {
		return fmt.Sprintf("%dy%dd", years, remainDays)
	}
	return fmt.Sprintf("%dy", years)
}

// --- nested access helpers ---

func nestedMap(obj map[string]interface{}, path ...string) (map[string]interface{}, bool) {
	val, ok := nestedValue(obj, path)
	if !ok {
		return nil, false
	}
	m, ok := val.(map[string]interface{})
	return m, ok
}

func nestedSlice(obj map[string]interface{}, path ...string) ([]interface{}, bool) {
	val, ok := nestedValue(obj, path)
	if !ok {
		return nil, false
	}
	s, ok := val.([]interface{})
	return s, ok
}

func intOr(obj map[string]interface{}, fallback int, path ...string) int {
	val, ok := nestedValue(obj, path)
	if !ok {
		return fallback
	}
	if i, ok := toInt(val); ok {
		return i
	}
	return fallback
}

func toInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int64:
		return int(v), true
	case int:
		return v, true
	case int32:
		return int(v), true
	}
	return 0, false
}
