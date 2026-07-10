package aiworkload

import (
	"github.com/stackrox/rox/pkg/k8sapi"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	inferenceServiceGV = schema.GroupVersion{Group: "serving.kserve.io", Version: "v1beta1"}

	inferenceServiceResources []k8sapi.APIResource

	InferenceService = registerInferenceServiceResource(v1.APIResource{
		Name:    "inferenceservices",
		Kind:    "InferenceService",
		Group:   inferenceServiceGV.Group,
		Version: inferenceServiceGV.Version,
	})
)

func GetInferenceServiceGV() schema.GroupVersion {
	return inferenceServiceGV
}

func GetInferenceServiceResources() []k8sapi.APIResource {
	return inferenceServiceResources
}

func registerInferenceServiceResource(resource v1.APIResource) k8sapi.APIResource {
	r := k8sapi.APIResource{
		APIResource: resource,
	}
	inferenceServiceResources = append(inferenceServiceResources, r)
	return r
}
