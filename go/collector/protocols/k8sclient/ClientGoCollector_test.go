package k8sclient

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saichler/l8pollaris/go/types/l8tpollaris"
	"github.com/saichler/l8srlz/go/serialize/object"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestParseGVR(t *testing.T) {
	gvr, err := ParseGVR("apps/v1/deployments")
	if err != nil {
		t.Fatalf("ParseGVR() error = %v", err)
	}
	if gvr.Group != "apps" || gvr.Version != "v1" || gvr.Resource != "deployments" {
		t.Fatalf("unexpected gvr: %#v", gvr)
	}

	gvr, err = ParseGVR("v1/pods")
	if err != nil {
		t.Fatalf("ParseGVR() error = %v", err)
	}
	if gvr.Group != "" || gvr.Version != "v1" || gvr.Resource != "pods" {
		t.Fatalf("unexpected core gvr: %#v", gvr)
	}
}

func TestParseCacheSpecDefaults(t *testing.T) {
	spec, err := ParseCacheSpec(`{"gvr":"v1/pods","nameFromArg":"name"}`, &l8tpollaris.L8Poll{})
	if err != nil {
		t.Fatalf("ParseCacheSpec() error = %v", err)
	}
	if spec.Mode != ModeGet {
		t.Fatalf("expected mode %s, got %s", ModeGet, spec.Mode)
	}
	if spec.Result != ResultMap {
		t.Fatalf("expected result %s, got %s", ResultMap, spec.Result)
	}
	if len(spec.Operations) != 3 {
		t.Fatalf("expected default operations, got %v", spec.Operations)
	}
}

func TestBuildCMapAndCTable(t *testing.T) {
	item := &CachedObject{
		GVR:       "v1/pods",
		Namespace: "default",
		Name:      "api-0",
		UID:       "123",
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name":      "api-0",
				"namespace": "default",
				"uid":       "123",
			},
			"status": map[string]interface{}{
				"phase": "Running",
			},
		},
	}

	cmap, err := BuildCMap(item, []string{"metadata.name", "status.phase"})
	if err != nil {
		t.Fatalf("BuildCMap() error = %v", err)
	}
	if len(cmap.Data) != 2 {
		t.Fatalf("expected 2 cmap fields, got %d", len(cmap.Data))
	}

	dec := object.NewDecode(cmap.Data["status.phase"], 0, nil)
	value, err := dec.Get()
	if err != nil {
		t.Fatalf("decode cmap value: %v", err)
	}
	if value.(string) != "Running" {
		t.Fatalf("expected Running, got %v", value)
	}

	tbl, err := BuildCTable([]*CachedObject{item}, []string{"metadata.name", "status.phase"}, []string{"NAME", "STATUS"})
	if err != nil {
		t.Fatalf("BuildCTable() error = %v", err)
	}
	if len(tbl.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(tbl.Rows))
	}
	row := tbl.Rows[0]
	if row == nil {
		t.Fatal("expected row 0 to exist")
	}
	dec = object.NewDecode(row.Data[1], 0, nil)
	value, err = dec.Get()
	if err != nil {
		t.Fatalf("decode table value: %v", err)
	}
	if value.(string) != "Running" {
		t.Fatalf("expected Running, got %v", value)
	}
}

func TestWebhookRulesFromPollarisModels(t *testing.T) {
	model := &l8tpollaris.L8Pollaris{
		Name: "kubernetesapi",
		Polling: map[string]*l8tpollaris.L8Poll{
			"pods": {
				Name:     "pods",
				Protocol: l8tpollaris.L8PProtocol_L8PKubernetesAPI,
				What:     `{"result":"table","mode":"list","gvr":"v1/pods"}`,
			},
		},
	}
	rules, err := WebhookRulesFromPollarisModels([]*l8tpollaris.L8Pollaris{model})
	if err != nil {
		t.Fatalf("WebhookRulesFromPollarisModels() error = %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	rule := rules[0]
	if rule.APIGroups[0] != "" || rule.APIVersions[0] != "v1" || rule.Resources[0] != "pods" {
		t.Fatalf("unexpected rule: %#v", rule)
	}
}

func TestAdmissionHandlerAllowsAndCallsBack(t *testing.T) {
	rules := []WebhookRule{{
		APIGroups:   []string{""},
		APIVersions: []string{"v1"},
		Resources:   []string{"pods"},
		Operations:  []string{"CREATE", "UPDATE", "DELETE"},
	}}
	called := false
	handler := NewAdmissionHandler(rules, func(event AdmissionEvent) error {
		called = true
		if event.Resource != "pods" || event.Name != "api-0" || event.Namespace != "default" {
			t.Fatalf("unexpected event: %#v", event)
		}
		return nil
	})
	obj := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": map[string]interface{}{
			"name":      "api-0",
			"namespace": "default",
		},
	}
	raw, _ := json.Marshal(obj)
	review := &admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			UID:       "123",
			Namespace: "default",
			Name:      "api-0",
			Operation: admissionv1.Create,
			Resource: metav1.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "pods",
			},
			Object: runtime.RawExtension{Raw: raw},
		},
	}
	body, _ := json.Marshal(review)
	req := httptest.NewRequest("POST", "/admission", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !called {
		t.Fatal("expected callback to be called")
	}
}
