package controller

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dungdev1/k8s-injector/pkg/config"
	"github.com/google/go-cmp/cmp"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const admissionReqFilePath = "../../docs/template/admission-request.json"

func TestAddNewConfig(t *testing.T) {
	req := admissionv1.AdmissionReview{}

	want := []PatchOperation{{
		Op:    "add",
		Path:  "/spec/containers/0/readinessProbe",
		Value: "{\"httpGet\":{\"path\":\"/healthz\",\"port\":3990},\"initialDelaySeconds\":5,\"timeoutSeconds\":5,\"periodSeconds\":10,\"successThreshold\":2,\"failureThreshold\":3}",
	}}

	byteValues, err := os.ReadFile(admissionReqFilePath)
	if err != nil {
		t.Errorf("Cannot read admission request template file %q", admissionReqFilePath)
	}
	json.Unmarshal(byteValues, &req)
	injConfig := map[string]*config.InjectionConfig{
		"/spec/containers/0/readinessProbe": {
			Readiness: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.IntOrString{Type: 0, IntVal: 3990},
					},
				},
				InitialDelaySeconds: 5,
				TimeoutSeconds:      5,
				PeriodSeconds:       10,
				SuccessThreshold:    2,
				FailureThreshold:    3,
			},
		},
	}

	namespaces := map[string]bool{
		"dbservice": true,
	}

	got, _, err := ApplyNewConfig(req.Request, injConfig, namespaces)
	if err != nil {
		t.Errorf("Apply new config failed")
	} else {
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("ApplyNewConfig() mismatch (-want +got):\n%s", diff)
		}
	}
}
