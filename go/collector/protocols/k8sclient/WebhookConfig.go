package k8sclient

import (
	"net/http"
	"sort"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const DefaultAdmissionPath = "/admission/kubernetes"

// HandlerRegistrar is the narrow registration surface needed from the REST server.
type HandlerRegistrar interface {
	RegisterHandler(path string, handler http.Handler)
}

// WebhookConfigOptions describes the deployment-facing webhook configuration.
type WebhookConfigOptions struct {
	Name        string
	ServiceName string
	Namespace   string
	Path        string
	FailureMode admissionregistrationv1.FailurePolicyType
}

// RegisterAdmissionHandler registers the collector admission endpoint on a web server.
func (c *ClientGoCollector) RegisterAdmissionHandler(registrar HandlerRegistrar, path string) error {
	if registrar == nil {
		return nil
	}
	if path == "" {
		path = DefaultAdmissionPath
	}
	handler, err := c.AdmissionHandler()
	if err != nil {
		return err
	}
	registrar.RegisterHandler(path, handler)
	return nil
}

// ValidatingWebhookYAML generates a ValidatingWebhookConfiguration from the active polls.
func ValidatingWebhookYAML(options WebhookConfigOptions) ([]byte, error) {
	rules, err := WebhookRulesFromBootModels()
	if err != nil {
		return nil, err
	}
	return ValidatingWebhookYAMLFromRules(options, rules)
}

func ValidatingWebhookYAMLFromRules(options WebhookConfigOptions, rules []WebhookRule) ([]byte, error) {
	if options.Path == "" {
		options.Path = DefaultAdmissionPath
	}
	if options.FailureMode == "" {
		options.FailureMode = admissionregistrationv1.Ignore
	}
	config := admissionregistrationFromRules(options.Name, options.ServiceName, options.Namespace, options.Path, rules)
	config.Webhooks[0].FailurePolicy = failurePolicyPtr(options.FailureMode)
	return yaml.Marshal(config)
}

func admissionregistrationFromRules(name, serviceName, namespace, path string, rules []WebhookRule) *admissionregistrationv1.ValidatingWebhookConfiguration {
	webhookName := serviceName + "." + namespace + ".svc"
	return &admissionregistrationv1.ValidatingWebhookConfiguration{
		TypeMeta:   typeMeta("admissionregistration.k8s.io/v1", "ValidatingWebhookConfiguration"),
		ObjectMeta: objectMeta(name),
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name:                    webhookName,
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             sideEffectsPtr(admissionregistrationv1.SideEffectClassNone),
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: namespace,
						Name:      serviceName,
						Path:      stringPtr(path),
					},
				},
				Rules: rulesToOperations(rules),
			},
		},
	}
}

func rulesToOperations(rules []WebhookRule) []admissionregistrationv1.RuleWithOperations {
	result := make([]admissionregistrationv1.RuleWithOperations, 0, len(rules))
	for _, rule := range rules {
		ops := make([]admissionregistrationv1.OperationType, 0, len(rule.Operations))
		sort.Strings(rule.Operations)
		for _, op := range rule.Operations {
			ops = append(ops, admissionregistrationv1.OperationType(op))
		}
		result = append(result, admissionregistrationv1.RuleWithOperations{
			Operations: ops,
			Rule: admissionregistrationv1.Rule{
				APIGroups:   rule.APIGroups,
				APIVersions: rule.APIVersions,
				Resources:   rule.Resources,
			},
		})
	}
	return result
}

func typeMeta(apiVersion, kind string) metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: apiVersion,
		Kind:       kind,
	}
}

func objectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name}
}

func stringPtr(value string) *string {
	return &value
}

func failurePolicyPtr(value admissionregistrationv1.FailurePolicyType) *admissionregistrationv1.FailurePolicyType {
	return &value
}

func sideEffectsPtr(value admissionregistrationv1.SideEffectClass) *admissionregistrationv1.SideEffectClass {
	return &value
}
