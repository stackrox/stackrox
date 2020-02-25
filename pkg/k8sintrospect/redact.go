package k8sintrospect

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RedactGeneric removes fields we don't care about for any object type. This includes:
// - the `kubectl.kubernetes.io/last-applied-configuration`
// - the selfLink field (this is not object-specific and fully captured by name, namespace, and kind)
// - the resourceVersion field (this is fully opaque with no reconstructible meaning)
func RedactGeneric(obj *unstructured.Unstructured) {
	annotations := obj.GetAnnotations()
	delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
	obj.SetAnnotations(annotations)
	obj.SetResourceVersion("")
	obj.SetSelfLink("")
}

// RedactSecret removes sensitive secret data from a secret object, but retains information about which keys are
// present.
func RedactSecret(secret *unstructured.Unstructured) {
	dataMap, found, err := unstructured.NestedMap(secret.UnstructuredContent(), "data")
	if found && err == nil {
		redactedStringData := make(map[string]string, len(dataMap))
		for key := range dataMap {
			redactedStringData[key] = "***REDACTED***"
		}
		_ = unstructured.SetNestedStringMap(secret.UnstructuredContent(), redactedStringData, "stringData")
	}
	unstructured.RemoveNestedField(secret.UnstructuredContent(), "data")
}

// FilterOutServiceAccountSecrets filters out secrets that are associated with a Kubernetes service account.
func FilterOutServiceAccountSecrets(secret *unstructured.Unstructured) bool {
	ty, _, _ := unstructured.NestedString(secret.Object, "type")
	return ty != `kubernetes.io/service-account-token`
}
