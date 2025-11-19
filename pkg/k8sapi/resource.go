package k8sapi

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// APIResource provides a wrapper around v1.APIResource.
type APIResource struct {
	v1.APIResource
}

// GroupVersionKind returns the GroupVersionKind which uniquely identifies the resource kind.
func (r *APIResource) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   r.Group,
		Version: r.Version,
		Kind:    r.Kind,
	}
}

// GroupVersionResource returns the GroupVersionResource which uniquely identifies the resource.
func (r *APIResource) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    r.Group,
		Version:  r.Version,
		Resource: r.Name,
	}
}
