package extensions

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/renderer"
	coreV1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralDBPasswordKey = `password`

	// canonicalCentralDBPasswordSecretName is the name of the secret that is mounted into Central (and Central DB).
	// This is not configurable; if a user specifies a different password secret, the password from that needs to be
	// mirrored into the canonical password secret.
	canonicalCentralDBPasswordSecretName = `central-db-password`
)

// ReconcileCentralDBPasswordExtension returns an extension that takes care of reconciling the central-db-password secret.
func ReconcileCentralDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(wrappedReconcileCentralDBPassword, client)
}

func wrappedReconcileCentralDBPassword(ctx context.Context, central *platform.Central, client ctrlClient.Client, _ func(statusFunc updateStatusFunc), _ logr.Logger) error {
	return reconcileCentralDBPassword(ctx, central, client)
}

func reconcileCentralDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client) error {

	var (
		err                        error
		hasReferencedSecret        bool
		referencedSecretName       string
		isExternalDB               bool
		centralDBPasswordSecretKey = ctrlClient.ObjectKey{Namespace: c.GetNamespace(), Name: canonicalCentralDBPasswordSecretName}
		password                   = renderer.CreatePassword()
	)

	if c.Spec.Central != nil {
		isExternalDB = c.Spec.Central.IsExternalDB()
	}

	if c.Spec.Central != nil && c.Spec.Central.DB != nil && c.Spec.Central.DB.PasswordSecret != nil {
		hasReferencedSecret = true
		referencedSecretName = c.Spec.Central.DB.PasswordSecret.Name
	}

	if !hasReferencedSecret && isExternalDB {
		return errors.New("spec.central.db.passwordSecret must be set when using an external database")
	}

	if hasReferencedSecret && len(referencedSecretName) == 0 {
		return errors.New("central.db.passwordSecret.name must be set")
	}

	if hasReferencedSecret {
		referencedSecretKey := ctrlClient.ObjectKey{Namespace: c.GetNamespace(), Name: referencedSecretName}
		password, err = obtainPasswordFromReferencedSecret(ctx, client, referencedSecretKey)
		if err != nil {
			return errors.Wrapf(err, "failed to get password from referenced secret %q", referencedSecretName)
		}

		// if the referenced secret name == central-db-password, we don't need to do anything.
		if referencedSecretName == canonicalCentralDBPasswordSecretName {
			return nil
		}
	}

	centralDBPasswordSecret := new(coreV1.Secret)
	err = client.Get(ctx, centralDBPasswordSecretKey, centralDBPasswordSecret)
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			// an unexpected error occurred while getting the secret
			return errors.Wrapf(err, "failed to get central-db-password secret %q", canonicalCentralDBPasswordSecretName)
		}
		// secret doesn't exist, create it
		centralDBPasswordSecret = makeNewCentralDBPasswordSecretWithPassword(c, password)
		err = client.Create(ctx, centralDBPasswordSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create central-db-password secret %q", canonicalCentralDBPasswordSecretName)
		}
	} else {
		// secret might need to be updated
		shouldUpdateOwnerReference := unsetCentralDBPasswordSecretOwnerReferenceIfNeeded(c, centralDBPasswordSecret)
		shouldUpdatePassword := updateCentralDBPasswordSecretDataIfNeeded(centralDBPasswordSecret, password, hasReferencedSecret)
		if shouldUpdateOwnerReference || shouldUpdatePassword {
			err = client.Update(ctx, centralDBPasswordSecret)
			if err != nil {
				return errors.Wrapf(err, "failed to update central-db-password secret %q", canonicalCentralDBPasswordSecretName)
			}
		}
	}

	return nil
}

func obtainPasswordFromReferencedSecret(ctx context.Context, client ctrlClient.Client, referencedSecretKey ctrlClient.ObjectKey) (password string, err error) {
	// get the referenced secret
	referencedSecret := new(coreV1.Secret)
	if err := client.Get(ctx, referencedSecretKey, referencedSecret); err != nil {
		return "", errors.Wrapf(err, "failed to get spec.central.db.passwordSecret %q", referencedSecretKey.Name)
	}

	// get the password from the referenced secret
	password, err = getAndValidatePasswordFromReferencedSecret(referencedSecret)
	if err != nil {
		return "", errors.Wrapf(err, "reading central db password from secret %s", canonicalCentralDBPasswordSecretName)
	}

	return password, nil
}

func updateCentralDBPasswordSecretDataIfNeeded(secret *coreV1.Secret, password string, hasReferencedSecret bool) bool {
	shouldUpdatePassword := false
	passwordIsEmpty := secret.Data == nil || len(secret.Data[centralDBPasswordKey]) == 0
	passwordsAreDifferent := secret.Data == nil || string(secret.Data[centralDBPasswordKey]) != password

	if passwordIsEmpty || hasReferencedSecret && passwordsAreDifferent {
		shouldUpdatePassword = true
	}

	if shouldUpdatePassword {
		secret.Data = map[string][]byte{
			centralDBPasswordKey: []byte(password),
		}
	}

	return shouldUpdatePassword
}

func unsetCentralDBPasswordSecretOwnerReferenceIfNeeded(c *platform.Central, secret *coreV1.Secret) bool {
	// make sure that the secret owner reference is unset. This is to ensure that PVCs which are not deleted
	// when Centrals are deleted do not have their passwords deleted.

	if len(secret.OwnerReferences) == 0 {
		return false
	}

	shouldUpdateOwnerReference := false
	centralOwnerRef := v1.NewControllerRef(c, c.GroupVersionKind())
	centralGK := c.GroupVersionKind().GroupKind()
	var resultOwnerRefs []v1.OwnerReference
	for _, ownerRef := range secret.OwnerReferences {
		ownerGV, err := schema.ParseGroupVersion(ownerRef.APIVersion)
		if err != nil {
			continue
		}
		ownerGK := ownerGV.WithKind(ownerRef.Kind).GroupKind()
		if ownerRef.UID == centralOwnerRef.UID &&
			ownerRef.Name == centralOwnerRef.Name &&
			ownerGK == centralGK {
			shouldUpdateOwnerReference = true
		} else {
			resultOwnerRefs = append(resultOwnerRefs, ownerRef)
		}
	}
	if shouldUpdateOwnerReference {
		secret.OwnerReferences = resultOwnerRefs
	}
	return shouldUpdateOwnerReference
}

func makeNewCentralDBPasswordSecretWithPassword(c *platform.Central, password string) *coreV1.Secret {
	// we do not set the owner reference, because this password is bound to the lifetime of the PVC which we might
	// not be managing. For security, we do not want to delete the password when the Central instance is deleted.
	return &coreV1.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:      canonicalCentralDBPasswordSecretName,
			Namespace: c.GetNamespace(),
		},
		Data: map[string][]byte{
			centralDBPasswordKey: []byte(password),
		},
	}
}

func getAndValidatePasswordFromReferencedSecret(secret *coreV1.Secret) (string, error) {
	if secret.Data == nil {
		return "", errors.Errorf("secret %q does not contain a %q entry", secret.Name, centralDBPasswordKey)
	}
	passwordBytes, ok := secret.Data[centralDBPasswordKey]
	if !ok {
		return "", errors.Errorf("secret %q does not contain a %q entry", secret.Name, centralDBPasswordKey)
	}
	password := strings.TrimSpace(string(passwordBytes))
	if len(password) == 0 {
		return "", errors.Errorf("secret %q contains an empty %q entry", secret.Name, centralDBPasswordKey)
	}
	if strings.ContainsAny(password, "\r\n") {
		return "", errors.Errorf("secret %q contains a multi-line %q entry", secret.Name, centralDBPasswordKey)
	}
	return password, nil
}
