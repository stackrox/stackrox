package utils

import (
	"context"

	"github.com/pkg/errors"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

// GetSecretWithUnstrucuteredObj gets a secret by using a unstrucutrued.Unstructured object.
// Using unstructured makes sure to don't use the default cache of the controller runtime client
func GetSecretWithUnstrucuteredObj(ctx context.Context, name string, namespace string, client ctrlClient.Client) (*coreV1.Secret, error) {
	secret := &coreV1.Secret{}
	unstructuredSecret := unstructured.Unstructured{}

	unstructuredSecret.SetKind("Secret")
	unstructuredSecret.SetAPIVersion(coreV1.SchemeGroupVersion.Version)
	key := ctrlClient.ObjectKey{Namespace: namespace, Name: name}

	err := client.Get(ctx, key, &unstructuredSecret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get unstructured secret")
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredSecret.Object, secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert unstrucutred secret to structured secret")
	}

	return secret, nil
}
