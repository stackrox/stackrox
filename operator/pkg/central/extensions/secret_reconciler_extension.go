package extensions

import (
	"context"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/utils"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	coreV1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type validateSecretDataFunc func(secretDataMap, bool) error
type generateSecretDataFunc func() (secretDataMap, error)

type secretReconciliationExtension struct {
	ctx        context.Context
	ctrlClient client.Client
	centralObj *platform.Central
}

func (r *secretReconciliationExtension) Namespace() string {
	return r.centralObj.Namespace
}

func (r *secretReconciliationExtension) reconcileSecret(name string, shouldExist bool, validate validateSecretDataFunc, generate generateSecretDataFunc, fixExisting bool) error {
	secret := &coreV1.Secret{}
	key := client.ObjectKey{Namespace: r.Namespace(), Name: name}
	if err := r.ctrlClient.Get(r.ctx, key, secret); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if !shouldExist {
		if secret == nil || !metav1.IsControlledBy(secret, r.centralObj) {
			return nil
		}

		if err := utils.DeleteExact(r.ctx, r.ctrlClient, secret); err != nil && !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "deleting %s secret", name)
		}
		return nil
	}

	if secret != nil {
		isManaged := metav1.IsControlledBy(secret, r.centralObj)
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
				*metav1.NewControllerRef(r.centralObj, r.centralObj.GroupVersionKind()),
			},
		},
		Data: data,
	}

	if secret == nil {
		if err := r.ctrlClient.Create(r.ctx, newSecret); err != nil {
			return errors.Wrapf(err, "creating new %s secret", name)
		}
	} else {
		newSecret.ResourceVersion = secret.ResourceVersion
		if err := r.ctrlClient.Update(r.ctx, newSecret); err != nil {
			return errors.Wrapf(err, "updating %s secret because existing instance failed validation", secret.Name)
		}
	}
	return nil
}
