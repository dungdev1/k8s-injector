package controller

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/dungdev1/k8s-injector/pkg/config"
	"github.com/rs/zerolog/log"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var podResource = metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}

var universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()

func ApplySecurity(req *admissionv1.AdmissionRequest) ([]PatchOperation, error) {
	log.Info().Msg("Apply Security to Pod")
	pod, err := decodePodResource(req)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("%+v\n", pod)

	return []PatchOperation{}, nil
}

func ApplyNewConfig(req *admissionv1.AdmissionRequest, injConfigs map[string]*config.InjectionConfig, namespaces map[string]bool) ([]PatchOperation, *bool, error) {
	log.Info().Msg("Applying new configs...")

	pod, err := decodePodResource(req)
	if err != nil {
		return nil, nil, err
	}
	if val, ok := pod.Labels["k8s-injection"]; ok && val == "disable" {
		log.Info().Msgf("does not apply configuration for pod %q because it's diabled", pod.Name)
		return []PatchOperation{}, nil, nil
	}

	log.Info().Msgf("Pod %q belong to namespace %q", pod.Name, req.Namespace)
	if !namespaces[req.Namespace] {
		log.Info().Msgf("This mutating webhook only support on Pod in namepsaces %v, add label k8s-injection=enabled to enable for namespace", namespaces)
		return []PatchOperation{}, nil, nil
	}

	getJsonObject := func(obj interface{}) (string, error) {
		val, err := json.Marshal(obj)
		if err != nil {
			return "", fmt.Errorf("could not marshal JSON patch value: %v", err)
		}
		return string(val), err
	}

	var patches []PatchOperation
	for name, cfg := range injConfigs {
		r := reflect.ValueOf(*cfg)
		// typeOfCfg := r.Type()
		// fmt.Printf("Num fields: %d\n", r.NumField())
		for i := 0; i < r.NumField(); i++ {
			// fieldName := typeOfCfg.Field(i).Name
			if r.Field(i).Kind() == reflect.Slice && r.Field(i).Len() == 0 {
				continue
			}
			if r.Field(i).Kind() == reflect.Ptr && r.Field(i).IsNil() {
				continue
			}
			// fmt.Printf("Field: %s - Kind: %s\n", fieldName, r.Field(i).Kind())
			// fmt.Println(r.Field(i).Interface())
			if r.Field(i).Kind() == reflect.Slice && string(name[len(name)-1]) == "-" {
				// Trường hợp add thêm 1 hoặc nhiều config vào list (đã có ít nhất 1 item)
				for j := 0; j < r.Field(i).Len(); j++ {
					val, err := getJsonObject(r.Field(i).Index(j).Interface())
					if err != nil {
						fmt.Println(err)
						continue
					}
					patches = append(patches, PatchOperation{
						Op:    "add",
						Path:  name,
						Value: val,
					})
				}
			} else {

				// Trường hợp add thêm 1 config/ 1 list config (new)
				val, err := getJsonObject(r.Field(i).Interface())
				if err != nil {
					fmt.Println(err)
					continue
				}
				patches = append(patches, PatchOperation{
					Op:    "add",
					Path:  name,
					Value: val,
				})
			}

		}
	}
	return patches, nil, nil
}

func decodePodResource(req *admissionv1.AdmissionRequest) (*corev1.Pod, error) {
	if req.Resource != podResource {
		return nil, fmt.Errorf("expect pod resource to be %s", podResource)
	}

	// Parse the Pod object
	raw := req.Object.Raw
	pod := corev1.Pod{}
	if _, _, err := universalDeserializer.Decode(raw, nil, &pod); err != nil {
		return nil, fmt.Errorf("could not deserialize pod object: %v", err)
	}
	fmt.Println(pod)

	return &pod, nil
}
