package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/dungdev1/k8s-injector/pkg/config"
	"github.com/rs/zerolog/log"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

const jsonContentType = `application/json`

type admitFunc func(*admissionv1.AdmissionRequest, map[string]*config.InjectionConfig, map[string]bool) ([]PatchOperation, error)

func isSystemNamespace(ns string) bool {
	return ns == metav1.NamespaceSystem || ns == metav1.NamespacePublic
}

// This function parses the HTTP request from admission webhook controller, and in case of a well-formed request
// , it call a admit function corresponding that implement logic for that request. The response will be returned as
// raw bytes
func HandleAdmitFunc(w http.ResponseWriter, r *http.Request, admit admitFunc, injConfigs map[string]*config.InjectionConfig, namespaces map[string]bool) ([]byte, error) {

	// Step 1: Request validation. Only handle POST requests with a body and json content type.
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("invalid method %s, only POST requests are allowed", r.Method)
	}

	if contentType := r.Header.Get("Content-Type"); contentType != jsonContentType {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("unsupported content type %s, only %s is supported", contentType, jsonContentType)
	}

	// Step 2: Parse the AdmissionReview request.
	var admissionReviewReq admissionv1.AdmissionReview

	// body, _ := ioutil.ReadAll(r.Body)
	// log.Info().Msgf("Admission request: \n%s", body)

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&admissionReviewReq)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, fmt.Errorf("could not deserialize request: %v", err)
	} else if admissionReviewReq.Request == nil {
		w.WriteHeader(http.StatusBadRequest)
		return nil, errors.New("malformed admission review: request is nil")
	}

	// Step 3: Construct the AdmissionReview response.
	admissionReviewResponse := admissionv1.AdmissionReview{
		Response: &admissionv1.AdmissionResponse{
			UID: admissionReviewReq.Request.UID,
		},
	}

	admissionReviewResponse.APIVersion = "admission.k8s.io/v1"
	admissionReviewResponse.Kind = "AdmissionReview"

	var patchOps []PatchOperation

	// Apply admit function only for non-system namespaces
	if !isSystemNamespace(admissionReviewReq.Request.Namespace) {
		patchOps, err = admit(admissionReviewReq.Request, injConfigs, namespaces)
	} else {
		log.Info().Msg("Just apply configuration only for non-system namespaces")
	}
	if err != nil {
		admissionReviewResponse.Response.Allowed = false
		admissionReviewResponse.Response.Result = &metav1.Status{
			Message: err.Error(),
		}
	} else {
		if len(patchOps) != 0 {
			patchBytes, err := json.Marshal(patchOps)
			log.Info().Msgf("%s", patchBytes)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return nil, fmt.Errorf("could not marshal JSON patch: %v", err)
			}
			admissionReviewResponse.Response.Patch = patchBytes
			var patchType admissionv1.PatchType = admissionv1.PatchTypeJSONPatch
			admissionReviewResponse.Response.PatchType = &patchType
		}
		admissionReviewResponse.Response.Allowed = true
	}

	// Return the AdmissionReview with a response as JSON
	bytes, err := json.Marshal(&admissionReviewResponse)
	if err != nil {
		return nil, fmt.Errorf("marshaling response: %v", err)
	}
	return bytes, nil
}
