package runner

import (
	"github.com/stackrox/rox/sensor/upgrader/common"
	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
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

func transferMetadata(newObjects []k8sobjects.Object, oldObjects map[k8sobjects.ObjectRef]k8sobjects.Object) {
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
