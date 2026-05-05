package k8sclient

import "fmt"

// handleResourceDeletion is the single entry point for all delete processing.
// Called by both the admission webhook handler and the informer DeleteFunc.
// It removes the object from cache, fires the onDelete callback (forwarding
// to parser/inventory), and triggers cascade logic for namespace and
// deployment deletions.
func handleResourceDeletion(gvrText, namespace, name string) {
	var uid string
	if existing, ok := shared.cache.Get(gvrText, namespace, name); ok {
		uid = existing.UID
	}

	shared.cache.Delete(gvrText, namespace, name)
	if shared.onDelete != nil {
		shared.onDelete(gvrText, namespace, name)
	}

	if gvrText == "v1/namespaces" {
		cascadeNamespaceDelete(name)
	}
	if gvrText == "apps/v1/deployments" {
		cascadeDeploymentDelete(namespace, name, uid)
	}
}

// cascadeNamespaceDelete removes all cached objects belonging to the
// given namespace. Called when a namespace DELETE event is received.
// This handles the case where namespace-scoped informers lose their
// watch stream before child resource DELETE events fire.
func cascadeNamespaceDelete(namespace string) {
	if namespace == "" {
		return
	}

	entries := shared.cache.List("", namespace, "")

	fmt.Printf("[CASCADE-NS] namespace=%s found=%d cached objects to delete\n",
		namespace, len(entries))

	for _, entry := range entries {
		if entry.GVR == "v1/namespaces" {
			continue
		}
		shared.cache.Delete(entry.GVR, entry.Namespace, entry.Name)
		if shared.onDelete != nil {
			shared.onDelete(entry.GVR, entry.Namespace, entry.Name)
		}
	}
}

// cascadeDeploymentDelete removes ReplicaSets and Pods owned by the
// deleted Deployment. K8s garbage collector deletes these asynchronously,
// but those DELETE events may be lost if the informer/webhook misses them.
func cascadeDeploymentDelete(namespace, deploymentName, deploymentUID string) {
	if namespace == "" || deploymentName == "" {
		return
	}

	allRS := shared.cache.List("apps/v1/replicasets", namespace, "")
	var ownedRS []*CachedObject
	for _, rs := range allRS {
		if isOwnedBy(rs, deploymentUID, deploymentName, "Deployment") {
			ownedRS = append(ownedRS, rs)
		}
	}

	fmt.Printf("[CASCADE-DEPLOY] deployment=%s/%s found=%d owned ReplicaSets\n",
		namespace, deploymentName, len(ownedRS))

	for _, rs := range ownedRS {
		rsUID := rs.UID
		rsName := rs.Name

		pods := shared.cache.List("v1/pods", namespace, "")
		podCount := 0
		for _, pod := range pods {
			if isOwnedBy(pod, rsUID, rsName, "ReplicaSet") {
				shared.cache.Delete(pod.GVR, pod.Namespace, pod.Name)
				if shared.onDelete != nil {
					shared.onDelete(pod.GVR, pod.Namespace, pod.Name)
				}
				podCount++
			}
		}
		fmt.Printf("[CASCADE-DEPLOY]   rs=%s/%s found=%d owned Pods\n",
			namespace, rsName, podCount)

		shared.cache.Delete(rs.GVR, rs.Namespace, rs.Name)
		if shared.onDelete != nil {
			shared.onDelete(rs.GVR, rs.Namespace, rs.Name)
		}
	}
}

func isOwnedBy(obj *CachedObject, ownerUID, ownerName, ownerKind string) bool {
	if obj == nil || obj.Object == nil {
		return false
	}
	metadataRaw, ok := obj.Object["metadata"]
	if !ok {
		return false
	}
	metadata, ok := metadataRaw.(map[string]interface{})
	if !ok {
		return false
	}
	ownersRaw, ok := metadata["ownerReferences"]
	if !ok {
		return false
	}
	owners, ok := ownersRaw.([]interface{})
	if !ok {
		return false
	}
	for _, ownerRaw := range owners {
		owner, ok := ownerRaw.(map[string]interface{})
		if !ok {
			continue
		}
		if ownerUID != "" {
			if uid, _ := owner["uid"].(string); uid == ownerUID {
				return true
			}
		}
		if name, _ := owner["name"].(string); name == ownerName {
			if kind, _ := owner["kind"].(string); kind == ownerKind {
				return true
			}
		}
	}
	return false
}
