package utils

import (
	"context"

	"github.com/pkg/errors"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RemoveOwnerRef removes an owner ref of the given owner object from the given object.
func RemoveOwnerRef(obj metav1.Object, owner metav1.Object) {
	r := obj.GetOwnerReferences()[:0]
	for _, v := range obj.GetOwnerReferences() {
		if v.UID == owner.GetUID() {
			continue
		}
		r = append(r, v)
	}
	obj.SetOwnerReferences(r)
}

// GetWithFallbackToUncached attempts to get a k8s object from the cached client
// if it does not find a matching object it will try to use a direct read
// to the k8s API server. This is necessary because controller-runtime returns
// 404 for all objects that don't match the cache selector.
func GetWithFallbackToUncached(ctx context.Context, client ctrlClient.Client, uncached ctrlClient.Reader, key ctrlClient.ObjectKey, obj ctrlClient.Object) error {
	if err := client.Get(ctx, key, obj); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "retrieving %s failed", key.Name)
		}
		return uncached.Get(ctx, key, obj)
	}

	return nil
}
