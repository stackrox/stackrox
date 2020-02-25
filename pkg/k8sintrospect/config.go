package k8sintrospect

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectConfig configures how objects of a Kubernetes resource are obtained.
type ObjectConfig struct {
	GVK           schema.GroupVersionKind
	LabelSelector *metav1.LabelSelector // Label selector to apply; MUST be set for non-namespaced resources.

	RedactionFunc func(obj *unstructured.Unstructured)
	FilterFunc    func(obj *unstructured.Unstructured) bool
}

// Config configures the behavior of the Kubernetes self-diagnosis feature.
type Config struct {
	Namespaces []string
	Objects    []ObjectConfig
	PathPrefix string
}
