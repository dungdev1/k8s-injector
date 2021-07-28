package admit

import (
	"fmt"
	"reflect"

	"github.com/dungdev1/k8s-injector/pkg/config"
	"github.com/dungdev1/k8s-injector/pkg/controller"
	"github.com/rs/zerolog/log"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var podResource = metav1.GroupVersionResource{Version: "v1", Resource: "pods"}

var universalDeserializer = serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()

func ApplySecurity(req *admissionv1.AdmissionRequest) ([]controller.PatchOperation, error) {
	log.Info().Msg("Apply Security to Pod")
	pod, err := decodePodResource(req)
	if err != nil {
		return nil, err
	}
	log.Debug().Msgf("%+v\n", pod)

	return []controller.PatchOperation{}, nil
}

func ApplyNewConfig(req *admissionv1.AdmissionRequest, injConfigs map[string]*config.InjectionConfig, namespaces map[string]bool) ([]controller.PatchOperation, error) {
	log.Info().Msg("Applying new configs...")

	pod, err := decodePodResource(req)
	if err != nil {
		return nil, err
	}
	if val, ok := pod.Labels["k8s-injection"]; ok && val == "disable" {
		log.Info().Msgf("Does not apply configuration for Replicas %q because it's diabled")
		return []controller.PatchOperation{}, nil
	}

	log.Debug().Msgf("ReplicaSet %q belong to namespace %q", pod.GenerateName, req.Namespace)
	if !namespaces[req.Namespace] {
		log.Info().Msgf("This mutating webhook only support on Pod in namepsaces %v, add label k8s-injection=enabled to enable for namespace", namespaces)
		return []controller.PatchOperation{}, nil
	}

	var patches []controller.PatchOperation
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
					patches = append(patches, controller.PatchOperation{
						Op:    "add",
						Path:  name,
						Value: r.Field(i).Index(j).Interface(),
					})
				}
			} else {

				// Trường hợp add thêm 1 config/ 1 list config (new)
				patches = append(patches, controller.PatchOperation{
					Op:    "add",
					Path:  name,
					Value: r.Field(i).Interface(),
				})
			}

		}
	}
	return patches, nil
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

	return &pod, nil
}
