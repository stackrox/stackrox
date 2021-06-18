package utils

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Deleter abstracts the Kubernetes Client deletion interface.
type Deleter interface {
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

// DeleteExact deletes exactly the given object. The deletion will fail if the target object
// no longer matches the specified obj w.r.t. uid and resource version.
func DeleteExact(ctx context.Context, deleter Deleter, obj metav1.Object) error {
	uid := obj.GetUID()
	resourceVersion := obj.GetResourceVersion()
	precond := metav1.Preconditions{
		UID:             &uid,
		ResourceVersion: &resourceVersion,
	}
	return deleter.Delete(ctx, obj.GetName(), metav1.DeleteOptions{Preconditions: &precond})
}
