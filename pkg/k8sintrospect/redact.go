package k8sintrospect

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const redactedValue = "***REDACTED***"

// sensitiveSecretAnnotations enumerates annotation keys whose values may contain
// sensitive data and therefore must be redacted before a secret is included in a
// diagnostic bundle. Unlike the secret `data` field, these values are not covered
// by the data redaction below.
var sensitiveSecretAnnotations = []string{
	// OpenShift stores a plaintext copy of the service account token in this
	// annotation on the generated dockercfg secrets (ROX-36824).
	"openshift.io/token-secret.value",
}

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
			redactedStringData[key] = redactedValue
		}
		_ = unstructured.SetNestedStringMap(secret.UnstructuredContent(), redactedStringData, "stringData")
	}
	unstructured.RemoveNestedField(secret.UnstructuredContent(), "data")

	// Some secrets carry sensitive values in their annotations (e.g. the plaintext
	// service account token OpenShift stores on dockercfg secrets). Redact those
	// values while keeping the annotation key, so the bundle still shows the
	// annotation was present.
	if annotations := secret.GetAnnotations(); len(annotations) > 0 {
		modified := false
		for _, key := range sensitiveSecretAnnotations {
			if _, ok := annotations[key]; ok {
				annotations[key] = redactedValue
				modified = true
			}
		}
		if modified {
			secret.SetAnnotations(annotations)
		}
	}
}

// FilterOutServiceAccountSecrets filters out secrets that are associated with a Kubernetes service account.
func FilterOutServiceAccountSecrets(secret *unstructured.Unstructured) bool {
	ty, _, _ := unstructured.NestedString(secret.Object, "type")
	return ty != `kubernetes.io/service-account-token`
}
