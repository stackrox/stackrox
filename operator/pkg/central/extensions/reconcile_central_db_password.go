package extensions

import (
	"context"
	"strings"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	commonExtensions "github.com/stackrox/rox/operator/pkg/common/extensions"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/pkg/renderer"
	coreV1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralDBPasswordKey = `password`

	defaultCentralDBPasswordSecretName = `central-db-password`
)

// ReconcileCentralDBPasswordExtension returns an extension that takes care of reconciling the central-db-password secret.
func ReconcileCentralDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralDBPassword, client)
}

func reconcileCentralDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, statusUpdater func(updateStatusFunc), log logr.Logger) error {
	run := &reconcileCentralDBPasswordExtensionRun{
		SecretReconciliator: commonExtensions.NewSecretReconciliator(client, c),
		centralObj:          c,
	}
	return run.Execute(ctx)
}

type reconcileCentralDBPasswordExtensionRun struct {
	*commonExtensions.SecretReconciliator
	centralObj *platform.Central
	password   string
}

func (r *reconcileCentralDBPasswordExtensionRun) readPasswordFromReferencedSecret(ctx context.Context) error {
	if r.centralObj.Spec.Central.DB.GetPasswordSecret() == nil {
		return nil
	}

	passwordSecretName := r.centralObj.Spec.Central.DB.PasswordSecret.Name

	passwordSecret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.centralObj.GetNamespace(), Name: passwordSecretName}
	if err := r.Client().Get(ctx, key, passwordSecret); err != nil {
		return errors.Wrapf(err, "failed to retrieve central db password secret %q", passwordSecretName)
	}

	password := strings.TrimSpace(string(passwordSecret.Data[centralDBPasswordKey]))
	if password == "" || strings.ContainsAny(password, "\r\n") {
		return errors.Errorf("central db password secret %s must contain a non-empty, single-line %q entry", passwordSecretName, centralDBPasswordKey)
	}

	r.password = password
	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) Execute(ctx context.Context) error {
	if r.centralObj.DeletionTimestamp != nil {
		return r.ReconcileSecret(ctx, defaultCentralDBPasswordSecretName, false, nil, nil, false)
	}

	if r.centralObj.Spec.Central.DB.GetPasswordGenerationDisabled() && r.centralObj.Spec.Central.DB.GetPasswordSecret() == nil {
		err := r.ReconcileSecret(ctx, defaultCentralDBPasswordSecretName, false, nil, nil, false)
		if err != nil {
			return err
		}
		return nil
	}

	if err := r.readPasswordFromReferencedSecret(ctx); err != nil {
		return err
	}
	// If the user puts the password into central-db-password secret then don't reconcile
	if secret := r.centralObj.Spec.Central.DB.GetPasswordSecret(); secret != nil && secret.Name == defaultCentralDBPasswordSecretName {
		return nil
	}
	if err := r.ReconcileSecret(ctx, defaultCentralDBPasswordSecretName, true, r.validateSecretData, r.generateDBPassword, true); err != nil {
		return errors.Wrap(err, "reconciling central-db-password secret")
	}
	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) validateSecretData(data types.SecretDataMap, controllerOwned bool) error {
	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) generateDBPassword() (types.SecretDataMap, error) {
	if r.password == "" {
		r.password = renderer.CreatePassword()
	}

	return types.SecretDataMap{
		centralDBPasswordKey: []byte(r.password),
	}, nil
}
