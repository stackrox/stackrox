package extensions

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// validateSecretDataFunc validates a secret to determine if the generation should run. The boolean parameter is true if the
// secret is managed by the running operator.
type validateSecretDataFunc func(types.SecretDataMap, bool) error
type generateSecretDataFunc func() (types.SecretDataMap, error)

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

// Namespace returns the namespace of the object the secret is owned by
func (r *SecretReconciliator) Namespace() string {
	return r.obj.GetNamespace()
}

// ReconcileSecret reconciles a secret with the given name, by making sure its existence matches "shouldExist" value.
// If the validateSecretDataFunc returns an error, then this function calls generateSecretDataFunc to get new secret data and updates the secret to "fix" it.
// If fixExisting is set to false, an already existing secret will never be overwritten. This is useful only when
// changing an invalid secret can bring more harm than good.
// In this case this function will return an error if "validate" rejects the secret.
// Also note that (regardless of the value of "fixExisting") this function will never touch a secret which is not owned by the object passed to the constructor.
func (r *SecretReconciliator) ReconcileSecret(ctx context.Context, name string, shouldExist bool, validate validateSecretDataFunc, generate generateSecretDataFunc, fixExisting bool) error {
	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.Namespace(), Name: name}
	if err := r.Client().Get(ctx, key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if !shouldExist {
		if secret == nil || !metav1.IsControlledBy(secret, r.obj) {
			return nil
		}

		if err := utils.DeleteExact(ctx, r.Client(), secret); err != nil && !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "deleting %s secret", name)
		}
		return nil
	}

	if secret != nil {
		isManaged := metav1.IsControlledBy(secret, r.obj)
		var validateErr error
		if validate != nil {
			validateErr = validate(secret.Data, isManaged)
		}

		if validateErr == nil {
			return nil // validation of existing secret successful - no reconciliation needed
		}
		// If the secret is unmanaged, we cannot fix it, so we should fail. The same applies if there is no
		// generate function specified, or if the caller told us not to attempt to fix it.
		if !isManaged || generate == nil || !fixExisting {
			return errors.Wrapf(validateErr, "existing %s secret is invalid, please delete the secret to allow fixing the issue", name)
		}
	}

	if generate == nil {
		return pkgUtils.ShouldErr(errors.Errorf("secret %s should exist, but no generation logic has been specified", name))
	}

	// Try to generate the secret, in order to fix it.
	data, err := generate()
	if err != nil {
		return errors.Wrapf(err, "generating data for new %s secret", name)
	}
	newSecret := &coreV1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: r.Namespace(),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.obj, r.obj.GroupVersionKind()),
			},
		},
		Data: data,
	}

	if secret == nil {
		if err := r.Client().Create(ctx, newSecret); err != nil {
			return errors.Wrapf(err, "creating new %s secret", name)
		}
	} else {
		newSecret.ResourceVersion = secret.ResourceVersion
		if err := r.Client().Update(ctx, newSecret); err != nil {
			return errors.Wrapf(err, "updating %s secret because existing instance failed validation", secret.Name)
		}
	}
	return nil
}
