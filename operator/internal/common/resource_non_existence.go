package common

import (
	"context"

	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Verifies that a resource identified by 'gvk', 'namespace' and 'name' does not exist already. Returns an error, if it does.
// The error returned comes from the apimachinery API errors and (unfortunately) requires a schema.GroupResource to be constructed.
// Hence we also need 'resource' in this function. This way we can keep it simple and require the resource name to be passed in
// by the caller instead of implementing dynamic discovery.
func VerifyResourceNonExistence(ctx context.Context, client ctrlClient.Reader, gvk schema.GroupVersionKind, resource, namespace, name string) error {
	key := ctrlClient.ObjectKey{Namespace: namespace, Name: name}
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(gvk)
	err := client.Get(ctx, key, u)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	gr := schema.GroupResource{
		Group:    gvk.Group,
		Resource: resource,
	}
	return apiErrors.NewAlreadyExists(gr, name)
}

// Returns true, if the (unstructured) CR has been reconciled before. To identify this case we check for the existence of 'status.productVersion',
// which is added on successful reconcilliation by this operator.
//
// Unfortunately we cannot simply check for the presence of the 'status' sub-resource, because the problematic helm-operator code, which does not
// include the CR kind in its keying for the Helm release, also causes the 'status.conditions' of two colliding CRs to be cross-contaminated.
func CustomResourceAlreadyReconciled(u *unstructured.Unstructured) bool {
	status, ok := u.Object["status"].(map[string]interface{})
	if !ok {
		return false
	}

	if status["productVersion"] == nil {
		return false
	}

	// No status.productVersion present on the CR -> we can assume that it
	// was never successfully reconciled by us.
	return true
}
