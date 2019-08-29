package plan

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/sensor/upgrader/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	serviceAccountGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "ServiceAccount",
	}
	serviceGVK = schema.GroupVersionKind{
		Version: "v1",
		Kind:    "Service",
	}
)

func clearServiceAccountDynamicFields(obj *unstructured.Unstructured) {
	// Remove the default token, as it is auto-populated.
	defaultTokenNamePrefix := fmt.Sprintf("%s-token-", obj.GetName())
	secrets, _, _ := unstructured.NestedSlice(obj.Object, "secrets")
	filteredSecrets := secrets[:0]
	for _, secret := range secrets {
		secretObj, _ := secret.(map[string]interface{})
		if secretName, _, _ := unstructured.NestedString(secretObj, "name"); !strings.HasPrefix(secretName, defaultTokenNamePrefix) {
			filteredSecrets = append(filteredSecrets, secret)
		}
	}
	_ = unstructured.SetNestedSlice(obj.Object, filteredSecrets, "secrets")
}

func clearServiceDynamicFields(obj *unstructured.Unstructured) {
	unstructured.RemoveNestedField(obj.Object, "spec", "clusterIP") // clusterIP may be dynamic
}

var (
	clearDynamicFieldsByGVK = map[schema.GroupVersionKind]func(*unstructured.Unstructured){
		serviceAccountGVK: clearServiceAccountDynamicFields,
		serviceGVK:        clearServiceDynamicFields,
	}
)

// normalizeObject heuristically clears dynamic fields to increase the likelihood of objects being recognized as equal.
func normalizeObject(obj *unstructured.Unstructured) {
	obj.SetUID("")
	obj.SetSelfLink("")
	obj.SetClusterName("")
	obj.SetCreationTimestamp(metav1.Time{})
	obj.SetResourceVersion("")
	obj.SetGeneration(0)
	delete(obj.GetAnnotations(), common.LastUpgradeIDAnnotationKey) // irrelevant for diff
	unstructured.RemoveNestedField(obj.Object, "status")

	if clearDynamicFieldsFn := clearDynamicFieldsByGVK[obj.GetObjectKind().GroupVersionKind()]; clearDynamicFieldsFn != nil {
		clearDynamicFieldsFn(obj)
	}
}
