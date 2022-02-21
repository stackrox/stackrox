package extensions

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/pkg/utils"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretDataMap represents data stored as part of a secret.
type SecretDataMap = map[string][]byte

type validateSecretDataFunc func(SecretDataMap, bool) error
type generateSecretDataFunc func() (SecretDataMap, error)

type k8sObject interface {
	metav1.Object
	schema.ObjectKind
}

// NewSecretReconciliationExtension creates a new SecretReconciliationExtension.
func NewSecretReconciliationExtension(ctx context.Context, client ctrlClient.Client, obj k8sObject) *SecretReconciliationExtension {
	return &SecretReconciliationExtension{
		ctx:    ctx,
		client: client,
		obj:    obj,
	}
}

// SecretReconciliationExtension reconciles a secret.
type SecretReconciliationExtension struct {
	ctx    context.Context
	client ctrlClient.Client
	obj    k8sObject
}

// Client returns the controller-runtime client used by the extension.
func (r *SecretReconciliationExtension) Client() ctrlClient.Client {
	return r.client
}

// Namespace returns the namespace of the object the secret is owned by
func (r *SecretReconciliationExtension) Namespace() string {
	return r.obj.GetNamespace()
}

// ReconcileSecret reconciles a secret with the given name. If the validateSecretDataFunc returns true the generateSecretDataFunc
// is called to return the secret data.
// fixExisting is set if an already existing secret should be fixed. This is not always the case, i.e. if a password or certificate
// often should not be overwritten.
func (r *SecretReconciliationExtension) ReconcileSecret(name string, shouldExist bool, validate validateSecretDataFunc, generate generateSecretDataFunc, fixExisting bool) error {
	secret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.Namespace(), Name: name}
	if err := r.Client().Get(r.ctx, key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if !shouldExist {
		if secret == nil || !metav1.IsControlledBy(secret, r.obj) {
			return nil
		}

		if err := utils.DeleteExact(r.ctx, r.Client(), secret); err != nil && !apiErrors.IsNotFound(err) {
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
		return pkgUtils.Should(errors.Errorf("secret %s should exist, but no generation logic has been specified", name))
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
		if err := r.Client().Create(r.ctx, newSecret); err != nil {
			return errors.Wrapf(err, "creating new %s secret", name)
		}
	} else {
		newSecret.ResourceVersion = secret.ResourceVersion
		if err := r.Client().Update(r.ctx, newSecret); err != nil {
			return errors.Wrapf(err, "updating %s secret because existing instance failed validation", secret.Name)
		}
	}
	return nil
}
