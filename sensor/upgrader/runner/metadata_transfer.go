package runner

import (
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func transferMetadataMap(oldMap, newMap map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range oldMap {
		if !common.ShouldTransferMetadataKey(k) {
			continue
		}
		result[k] = v
	}
	for k, v := range newMap {
		result[k] = v
	}
	return result
}

func transferMetadata(newObjects []*unstructured.Unstructured, oldObjects map[k8sobjects.ObjectRef]*unstructured.Unstructured) {
	for _, newObj := range newObjects {
		newObjRef := k8sobjects.RefOf(newObj)
		oldObj := oldObjects[newObjRef]
		if oldObj == nil {
			continue
		}

		newObj.SetLabels(transferMetadataMap(oldObj.GetLabels(), newObj.GetLabels()))
		newObj.SetAnnotations(transferMetadataMap(oldObj.GetAnnotations(), newObj.GetAnnotations()))
	}
}
