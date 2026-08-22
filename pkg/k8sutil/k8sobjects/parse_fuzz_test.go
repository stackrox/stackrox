package k8sobjects

import (
	"testing"
)

// FuzzParseRef ensures ParseRef doesn't panic on arbitrary input strings.
// It tests the parser against various malformed and edge-case inputs.
func FuzzParseRef(f *testing.F) {
	// Seed corpus with valid examples covering different formats

	// Valid cases: namespaced resources
	f.Add("Deployment:apps/v1:default/nginx")
	f.Add("Pod:v1:kube-system/coredns")
	f.Add("Service:v1:my-namespace/my-service")
	f.Add("ConfigMap:v1:test-ns/config")

	// Valid cases: cluster-scoped resources (no namespace)
	f.Add("ClusterRole:rbac.authorization.k8s.io/v1:admin")
	f.Add("Namespace:v1:default")
	f.Add("Node:v1:worker-1")
	f.Add("PersistentVolume:v1:pv-001")

	// Valid cases: custom resources with group
	f.Add("CustomResource:example.com/v1alpha1:default/my-cr")
	f.Add("NetworkPolicy:networking.k8s.io/v1:prod/allow-all")

	// Edge cases: empty parts (should error, not panic)
	f.Add("::")
	f.Add("::name")
	f.Add("Kind::")
	f.Add(":group/version:")
	f.Add("::namespace/name")

	// Edge cases: missing colons (should error, not panic)
	f.Add("")
	f.Add("no-colons")
	f.Add("only:one")
	f.Add("too:many:colons:here")

	// Edge cases: missing slashes
	f.Add("Kind:version:name")
	f.Add("Kind:nogroup:default/name")

	// Edge cases: too many slashes
	f.Add("Kind:group/version:ns/sub/name")
	f.Add("Kind:group/v1:a/b/c/d")

	// Edge cases: special characters
	f.Add("Kind-123:apps/v1:test-ns/name-with-dashes")
	f.Add("Kind_underscore:v1:name_with_underscore")
	f.Add("Kind.dot:v1:namespace.with.dots/name.with.dots")

	// Edge cases: whitespace
	f.Add(" : : ")
	f.Add("Kind :group/v1:name")
	f.Add("Kind: group/v1 :name")

	// Edge cases: unicode
	f.Add("Kind:v1:namespace/名前")
	f.Add("Deployment:apps/v1:default/nginx-🚀")

	// Edge cases: very long strings
	longStr := "VeryLongKind"
	for range 10 {
		longStr += longStr
	}
	f.Add(longStr + ":v1:name")

	f.Fuzz(func(t *testing.T, input string) {
		// The main assertion: ParseRef must not panic
		ref, err := ParseRef(input)

		// If parsing succeeded, just exercise the result to catch panics
		if err == nil {
			_ = ref.GVK.Kind
			_ = ref.GVK.Group
			_ = ref.GVK.Version
			_ = ref.Namespace
			_ = ref.Name
		}

		// If parsing failed, that's fine - we just verify no panic occurred
		// The function has already returned, so if we reach here, no panic happened
	})
}
