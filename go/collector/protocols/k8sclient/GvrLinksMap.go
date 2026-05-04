package k8sclient

// gvrToLinksId maps GVR strings (as used by informers) to LinksId values
// (as defined in probler/prob/common/Links_k8s.go and parser/boot/k8s/).
var gvrToLinksId = map[string]string{
	"v1/pods":                      "K8sPod",
	"apps/v1/deployments":          "K8sDeploy",
	"apps/v1/statefulsets":         "K8sSts",
	"apps/v1/daemonsets":           "K8sDs",
	"apps/v1/replicasets":          "K8sRs",
	"batch/v1/jobs":                "K8sJob",
	"batch/v1/cronjobs":            "K8sCj",
	"v1/services":                  "K8sSvc",
	"v1/configmaps":                "K8sCm",
	"v1/secrets":                   "K8sSec",
	"v1/namespaces":                "K8sNs",
	"v1/nodes":                     "K8sNode",
	"v1/persistentvolumes":         "K8sPv",
	"v1/persistentvolumeclaims":    "K8sPvc",
	"v1/endpoints":                 "K8sEp",
	"v1/events":                    "K8sEvt",
	"v1/serviceaccounts":           "K8sSa",
	"v1/limitranges":               "K8sLr",
	"v1/resourcequotas":            "K8sRq",

	"networking.k8s.io/v1/ingresses":       "K8sIng",
	"networking.k8s.io/v1/ingressclasses":  "K8sIngCl",
	"networking.k8s.io/v1/networkpolicies": "K8sNetPol",
	"policy/v1/poddisruptionbudgets":       "K8sPdb",
	"storage.k8s.io/v1/storageclasses":     "K8sScl",
	"autoscaling/v2/horizontalpodautoscalers":           "K8sHpa",
	"discovery.k8s.io/v1/endpointslices":                "K8sEpSl",
	"rbac.authorization.k8s.io/v1/roles":                "K8sRole",
	"rbac.authorization.k8s.io/v1/rolebindings":         "K8sRb",
	"rbac.authorization.k8s.io/v1/clusterroles":         "K8sCr",
	"rbac.authorization.k8s.io/v1/clusterrolebindings":  "K8sCrb",
	"apiextensions.k8s.io/v1/customresourcedefinitions": "K8sCrd",

	"management.loft.sh/v1/virtualclusterinstances": "K8sVCl",

	"networking.istio.io/v1alpha3/envoyfilters":            "IstioEf",
	"networking.istio.io/v1beta1/virtualservices":          "IstioVs",
	"networking.istio.io/v1beta1/destinationrules":         "IstioDr",
	"networking.istio.io/v1beta1/gateways":                 "IstioGw",
	"networking.istio.io/v1beta1/serviceentries":           "IstioSe",
	"networking.istio.io/v1beta1/sidecars":                 "IstioSc",
	"security.istio.io/v1beta1/authorizationpolicies":      "IstioAp",
	"security.istio.io/v1beta1/peerauthentications":        "IstioPa",
}

func GVRToLinksId(gvrText string) string {
	return gvrToLinksId[gvrText]
}
