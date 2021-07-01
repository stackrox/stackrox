package extensions

import (
	"context"

	"github.com/pkg/errors"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/utils"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type validateSecretDataFunc func(secretDataMap, bool) error
type generateSecretDataFunc func() (secretDataMap, error)

type secretReconciliationExtension struct {
	ctx        context.Context
	k8sClient  kubernetes.Interface
	centralObj *centralv1Alpha1.Central
}

func (r *secretReconciliationExtension) Namespace() string {
	return r.centralObj.Namespace
}

func (r *secretReconciliationExtension) SecretsClient() coreV1.SecretInterface {
	return r.k8sClient.CoreV1().Secrets(r.Namespace())
}

func (r *secretReconciliationExtension) reconcileSecret(name string, shouldExist bool, validate validateSecretDataFunc, generate generateSecretDataFunc) error {
	secretsClient := r.SecretsClient()

	secret, err := secretsClient.Get(r.ctx, name, metav1.GetOptions{})
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "checking existence of %s secret", name)
		}
		secret = nil
	}
	if !shouldExist {
		if secret == nil || !metav1.IsControlledBy(secret, r.centralObj) {
			return nil
		}

		if err := utils.DeleteExact(r.ctx, secretsClient, secret); err != nil && !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "deleting %s secret", name)
		}
		return nil
	}

	if secret != nil {
		if validate != nil {
			if err := validate(secret.Data, metav1.IsControlledBy(secret, r.centralObj)); err != nil {
				return errors.Wrapf(err, "validating existing %s secret", name)
			}
		}
		return nil
	}

	if generate == nil {
		return pkgUtils.Should(errors.Errorf("secret %s should exist, but no generation logic has been specified", name))
	}
	data, err := generate()
	if err != nil {
		return errors.Wrapf(err, "generating data for new %s secret", name)
	}
	newSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(r.centralObj, r.centralObj.GroupVersionKind()),
			},
		},
		Data: data,
	}
	if _, err := secretsClient.Create(r.ctx, newSecret, metav1.CreateOptions{}); err != nil {
		return errors.Wrapf(err, "creating new %s secret", name)
	}
	return nil
}
