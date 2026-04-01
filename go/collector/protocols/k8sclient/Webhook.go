package k8sclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/saichler/l8parser/go/parser/boot"
	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
)

type WebhookRule struct {
	APIGroups   []string
	APIVersions []string
	Resources   []string
	Operations  []string
}

type AdmissionEvent struct {
	UID         string
	Group       string
	Version     string
	Resource    string
	SubResource string
	Namespace   string
	Name        string
	Operation   string
	Object      *unstructured.Unstructured
	OldObject   *unstructured.Unstructured
}

type AdmissionCallback func(AdmissionEvent) error

func WebhookRulesFromBootModels() ([]WebhookRule, error) {
	return WebhookRulesFromPollarisModels(boot.GetAllPolarisModels())
}

func WebhookRulesFromPollarisModels(models []*l8tpollaris.L8Pollaris) ([]WebhookRule, error) {
	agg := make(map[string]*WebhookRule)
	for _, model := range models {
		if model == nil {
			continue
		}
		for _, poll := range model.Polling {
			if poll == nil || poll.Protocol != l8tpollaris.L8PProtocol_L8PKubernetesAPI {
				continue
			}
			spec, err := ParseCacheSpec(poll.What, poll)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", model.Name, poll.Name, err)
			}
			gvr, err := ParseGVR(spec.GVR)
			if err != nil {
				return nil, fmt.Errorf("%s/%s: %w", model.Name, poll.Name, err)
			}
			key := gvr.Group + "|" + gvr.Version + "|" + gvr.Resource
			rule, ok := agg[key]
			if !ok {
				rule = &WebhookRule{
					APIGroups:   []string{gvr.Group},
					APIVersions: []string{gvr.Version},
					Resources:   []string{gvr.Resource},
					Operations:  []string{},
				}
				agg[key] = rule
			}
			for _, op := range spec.Operations {
				addUniqueString(&rule.Operations, op)
			}
		}
	}

	keys := make([]string, 0, len(agg))
	for key := range agg {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]WebhookRule, 0, len(keys))
	for _, key := range keys {
		rule := agg[key]
		sort.Strings(rule.Operations)
		result = append(result, *rule)
	}
	return result, nil
}

// NewAdmissionHandler returns an HTTP handler for Kubernetes admission reviews.
// The handler always allows the request (Allowed: true) — it is used purely
// for observation and cache population, never for blocking mutations.
func NewAdmissionHandler(rules []WebhookRule, callback AdmissionCallback) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		review := &admissionv1.AdmissionReview{}
		if err = json.Unmarshal(data, review); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		response := &admissionv1.AdmissionResponse{Allowed: true}
		if review.Request != nil {
			response.UID = review.Request.UID
			if matchesWebhookRule(rules, review.Request) && callback != nil {
				event := buildAdmissionEvent(review.Request)
				if err = callback(event); err != nil {
					response.Warnings = []string{err.Error()}
					response.Result = &metav1.Status{
						Status:  "Success",
						Message: err.Error(),
					}
				}
			}
		}
		out := &admissionv1.AdmissionReview{
			TypeMeta: review.TypeMeta,
			Response: response,
		}
		payload, err := json.Marshal(out)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})
}

func matchesWebhookRule(rules []WebhookRule, request *admissionv1.AdmissionRequest) bool {
	if request == nil {
		return false
	}
	op := string(request.Operation)
	for _, rule := range rules {
		if !containsString(rule.APIGroups, request.Resource.Group) {
			continue
		}
		if !containsString(rule.APIVersions, request.Resource.Version) {
			continue
		}
		if !containsString(rule.Resources, request.Resource.Resource) {
			continue
		}
		if !containsString(rule.Operations, op) {
			continue
		}
		return true
	}
	return false
}

func buildAdmissionEvent(request *admissionv1.AdmissionRequest) AdmissionEvent {
	event := AdmissionEvent{
		UID:         string(request.UID),
		Group:       request.Resource.Group,
		Version:     request.Resource.Version,
		Resource:    request.Resource.Resource,
		SubResource: request.SubResource,
		Namespace:   request.Namespace,
		Name:        request.Name,
		Operation:   string(request.Operation),
	}
	if obj, ok := decodeAdmissionObject(request.Object.Raw); ok {
		event.Object = obj
		if event.Namespace == "" {
			event.Namespace = obj.GetNamespace()
		}
		if event.Name == "" {
			event.Name = obj.GetName()
		}
	}
	if obj, ok := decodeAdmissionObject(request.OldObject.Raw); ok {
		event.OldObject = obj
	}
	return event
}

func decodeAdmissionObject(raw []byte) (*unstructured.Unstructured, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	item := &unstructured.Unstructured{}
	if err := json.Unmarshal(raw, item); err != nil {
		return nil, false
	}
	return item, true
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func addUniqueString(items *[]string, value string) {
	if containsString(*items, value) {
		return
	}
	*items = append(*items, value)
}
