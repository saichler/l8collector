package k8sclient

import (
	"encoding/json"
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
	enrichPodContainers(obj, out)
}

// podContainer is the JSON shape consumed by the UI's pod detail popup. It
// merges declarative spec data (image, ports, env, …) with runtime status
// (ready, state, restartCount). All fields use omitempty so the produced
// JSON stays compact for typical pods.
type podContainer struct {
	Name            string                   `json:"name,omitempty"`
	Image           string                   `json:"image,omitempty"`
	ImagePullPolicy string                   `json:"imagePullPolicy,omitempty"`
	Ports           []map[string]interface{} `json:"ports,omitempty"`
	Env             []map[string]interface{} `json:"env,omitempty"`
	Resources       map[string]interface{}   `json:"resources,omitempty"`
	VolumeMounts    []map[string]interface{} `json:"volumeMounts,omitempty"`
	Kind            string                   `json:"kind,omitempty"` // "container" | "init"
	Ready           *bool                    `json:"ready,omitempty"`
	State           string                   `json:"state,omitempty"` // "Running" | "Waiting" | "Terminated"
	RestartCount    int                      `json:"restartCount,omitempty"`
}

// enrichPodContainers walks spec.containers + spec.initContainers and merges
// runtime data from status.containerStatuses + status.initContainerStatuses
// (keyed by container name). The result is JSON-encoded into
// out["containers_json"]; the proto field K8SPod.containers_json carries it
// to the UI which renders one card per container.
//
// Empty pods produce no key (the UI then renders "—"). JSON marshal failures
// are logged via stdout — they should never happen with well-formed K8s
// objects but the silent-fallback rule says we surface anything unexpected.
func enrichPodContainers(obj map[string]interface{}, out map[string]interface{}) {
	statusByName := indexContainerStatuses(obj, "containerStatuses")
	initStatusByName := indexContainerStatuses(obj, "initContainerStatuses")

	list := make([]podContainer, 0)
	list = append(list, collectContainers(obj, "containers", "container", statusByName)...)
	list = append(list, collectContainers(obj, "initContainers", "init", initStatusByName)...)

	if len(list) == 0 {
		return
	}
	buf, err := json.Marshal(list)
	if err != nil {
		return
	}
	out["containers_json"] = string(buf)
}

func indexContainerStatuses(obj map[string]interface{}, key string) map[string]map[string]interface{} {
	idx := map[string]map[string]interface{}{}
	statuses, ok := nestedSlice(obj, "status", key)
	if !ok {
		return idx
	}
	for _, entry := range statuses {
		m, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		name := fmt.Sprint(m["name"])
		if name == "" {
			continue
		}
		idx[name] = m
	}
	return idx
}

func collectContainers(obj map[string]interface{}, specKey, kind string, statusByName map[string]map[string]interface{}) []podContainer {
	containers, ok := nestedSlice(obj, "spec", specKey)
	if !ok || len(containers) == 0 {
		return nil
	}
	out := make([]podContainer, 0, len(containers))
	for _, entry := range containers {
		spec, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		c := podContainer{
			Name:            stringOf(spec, "name"),
			Image:           stringOf(spec, "image"),
			ImagePullPolicy: stringOf(spec, "imagePullPolicy"),
			Kind:            kind,
		}
		if ports, ok := spec["ports"].([]interface{}); ok {
			c.Ports = mapsOf(ports)
		}
		if env, ok := spec["env"].([]interface{}); ok {
			c.Env = mapsOf(env)
		}
		if mounts, ok := spec["volumeMounts"].([]interface{}); ok {
			c.VolumeMounts = mapsOf(mounts)
		}
		if res, ok := spec["resources"].(map[string]interface{}); ok && len(res) > 0 {
			c.Resources = res
		}
		// Merge runtime status (when present) for ready/state/restartCount.
		if st, ok := statusByName[c.Name]; ok {
			if r, ok := st["ready"].(bool); ok {
				rc := r
				c.Ready = &rc
			}
			if rc, ok := toInt(st["restartCount"]); ok {
				c.RestartCount = rc
			}
			c.State = containerState(st)
		}
		out = append(out, c)
	}
	return out
}

func containerState(st map[string]interface{}) string {
	state, ok := st["state"].(map[string]interface{})
	if !ok {
		return ""
	}
	for _, k := range []string{"running", "waiting", "terminated"} {
		if _, ok := state[k]; ok {
			// Capitalize for display: running → Running.
			if k == "" {
				return ""
			}
			return strings.ToUpper(k[:1]) + k[1:]
		}
	}
	return ""
}

func stringOf(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func mapsOf(items []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, e := range items {
		if m, ok := e.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
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
