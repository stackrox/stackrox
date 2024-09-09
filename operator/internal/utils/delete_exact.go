package utils

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Deleter abstracts the controller Client deletion interface.
type Deleter interface {
	Delete(ctx context.Context, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error
}

// DeleteExact deletes exactly the given object. The deletion will fail if the target object
// no longer matches the specified obj w.r.t. uid and resource version.
func DeleteExact(ctx context.Context, deleter Deleter, obj ctrlClient.Object) error {
	uid := obj.GetUID()
	resourceVersion := obj.GetResourceVersion()
	precond := metav1.Preconditions{
		UID:             &uid,
		ResourceVersion: &resourceVersion,
	}
	return deleter.Delete(ctx, obj, &ctrlClient.DeleteOptions{Preconditions: &precond})
}
