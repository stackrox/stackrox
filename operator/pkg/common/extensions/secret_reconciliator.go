package extensions

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// validateSecretDataFunc validates a secret to determine if the generation should run. The boolean parameter is true if the
// secret is managed by the running operator.
type validateSecretDataFunc func(types.SecretDataMap, bool) error

// generateSecretDataFunc generates new content of a secret.
// The input data map contains the pre-existing secret content (if any) - in rare cases
// it is needed in order to preserve selected fields rather than regenerate them.
type generateSecretDataFunc func(types.SecretDataMap) (types.SecretDataMap, error)

// NewSecretReconciliator creates a new SecretReconciliator. It takes a context and controller client.
// The obj parameter is the owner object (i.e. a custom resource).
func NewSecretReconciliator(client ctrlClient.Client, obj types.K8sObject) *SecretReconciliator {
	return &SecretReconciliator{
		client: client,
		obj:    obj,
	}
}

// SecretReconciliator reconciles a secret.
type SecretReconciliator struct {
	client ctrlClient.Client
	obj    types.K8sObject
}

// Client returns the controller-runtime client used by the extension.
func (r *SecretReconciliator) Client() ctrlClient.Client {
	return r.client
}

// DeleteSecret makes sure a secret with the given name does NOT exist.
// NOTE that this function will never touch a secret which is not owned by the object passed to the constructor.
func (r *SecretReconciliator) DeleteSecret(ctx context.Context, name string) error {
	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.obj.GetNamespace(), Name: name}
	if err := r.Client().Get(ctx, key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if secret == nil || !metav1.IsControlledBy(secret, r.obj) {
		return nil
	}

	if err := utils.DeleteExact(ctx, r.Client(), secret); err != nil && !apiErrors.IsNotFound(err) {
		return errors.Wrapf(err, "deleting secret %s", key)
	}
	return nil
}

// EnsureSecret makes sure a secret with the given name exists.
// If the validateSecretDataFunc returns an error, then this function calls generateSecretDataFunc to get new secret data and updates the secret to "fix" it.
// Also note that this function will refuse to touch a secret which is not owned by the object passed to the constructor.
func (r *SecretReconciliator) EnsureSecret(ctx context.Context, name string, validate validateSecretDataFunc, generate generateSecretDataFunc) error {
	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.obj.GetNamespace(), Name: name}
	if err := r.Client().Get(ctx, key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}

	var oldData types.SecretDataMap
	var validateErr error
	if secret != nil {
		isManaged := metav1.IsControlledBy(secret, r.obj)
		validateErr = validate(secret.Data, isManaged)

		if validateErr == nil {
			return nil // validation of existing secret successful - no reconciliation needed
		}
		// If the secret is unmanaged, we cannot fix it, so we should fail.
		if !isManaged {
			return errors.Wrapf(validateErr,
				"existing %s secret is invalid (%s), but not owned by the CR, please delete the secret to allow fixing the issue",
				validateErr.Error(), name)
		}
		oldData = secret.Data
	}

	// Try to generate the secret, in order to fix it.
	data, err := generate(oldData)
	if err != nil {
		extraInfo := "new"
		if validateErr != nil {
			extraInfo = fmt.Sprintf("invalid (%s)", validateErr.Error())
		}
		return errors.Wrapf(err, "error generating data for %s %s secret", extraInfo, name)
	}
	newSecret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.obj.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.obj, r.obj.GroupVersionKind()),
			},
		},
		Data: data,
	}

	if secret == nil {
		if err := r.Client().Create(ctx, newSecret); err != nil {
			return errors.Wrapf(err, "creating new %s secret failed", name)
		}
	} else {
		newSecret.ResourceVersion = secret.ResourceVersion
		if err := r.Client().Update(ctx, newSecret); err != nil {
			return errors.Wrapf(err, "updating invalid %s secret (%s) failed", secret.Name, validateErr)
		}
	}
	return nil
}
