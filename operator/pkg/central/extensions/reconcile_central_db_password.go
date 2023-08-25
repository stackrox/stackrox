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

	// canonicalCentralDBPasswordSecretName is the name of the secret that is mounted into Central (and Central DB).
	// This is not configurable; if a user specifies a different password secret, the password from that needs to be
	// mirrored into the canonical password secret.
	canonicalCentralDBPasswordSecretName = `central-db-password`
)

// ReconcileCentralDBPasswordExtension returns an extension that takes care of reconciling the central-db-password secret.
func ReconcileCentralDBPasswordExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileCentralDBPassword, client)
}

func reconcileCentralDBPassword(ctx context.Context, c *platform.Central, client ctrlClient.Client, _ func(updateStatusFunc), _ logr.Logger) error {
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

func (r *reconcileCentralDBPasswordExtensionRun) readAndSetPasswordFromReferencedSecret(ctx context.Context) error {
	if r.centralObj.Spec.Central.DB.GetPasswordSecret() == nil {
		return errors.New("no password secret was specified in spec.central.db.passwordSecret")
	}

	passwordSecretName := r.centralObj.Spec.Central.DB.PasswordSecret.Name

	passwordSecret := &coreV1.Secret{}
	key := ctrlClient.ObjectKey{Namespace: r.centralObj.GetNamespace(), Name: passwordSecretName}
	if err := r.Client().Get(ctx, key, passwordSecret); err != nil {
		return errors.Wrapf(err, "failed to retrieve central db password secret %q", passwordSecretName)
	}

	password, err := passwordFromSecretData(passwordSecret.Data)
	if err != nil {
		return errors.Wrapf(err, "reading central db password from secret %s", passwordSecretName)
	}

	r.password = password
	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) Execute(ctx context.Context) error {
	if r.centralObj.DeletionTimestamp != nil {
		return r.ReconcileSecret(ctx, canonicalCentralDBPasswordSecretName, false, nil, nil, false)
	}

	centralSpec := r.centralObj.Spec.Central
	if centralSpec != nil && centralSpec.DB != nil {
		dbSpec := centralSpec.DB
		dbPasswordSecret := dbSpec.PasswordSecret
		if dbSpec.IsExternal() && dbPasswordSecret == nil {
			return errors.New("setting spec.central.db.passwordSecret is mandatory when using an external DB")
		}

		if dbPasswordSecret != nil {
			if err := r.readAndSetPasswordFromReferencedSecret(ctx); err != nil {
				return err
			}
			// If the user wants to use the central-db-password secret directly, that's fine, and we don't have anything more to do.
			if dbPasswordSecret.Name == canonicalCentralDBPasswordSecretName {
				return nil
			}
		}
	}

	// At this point, r.password was set via readAndSetPasswordFromReferencedSecret above (user-specified mode), or is unset,
	// in which case the auto-generation logic will take effect.
	if err := r.ReconcileSecret(ctx, canonicalCentralDBPasswordSecretName, true, r.validateSecretData, r.generateDBPassword, true); err != nil {
		return errors.Wrapf(err, "reconciling %s secret", canonicalCentralDBPasswordSecretName)
	}
	return nil
}

func (r *reconcileCentralDBPasswordExtensionRun) validateSecretData(data types.SecretDataMap, _ bool) error {
	password, err := passwordFromSecretData(data)
	if err != nil {
		return errors.Wrap(err, "validating existing secret data")
	}
	if r.password != "" && r.password != password {
		return errors.New("existing password does not match expected one")
	}
	// The following assignment shouldn't have any consequences, as a successful validation should prevent generation
	// from being invoked, but better safe than sorry (about clobbering a user-set password).
	r.password = password
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

func passwordFromSecretData(data types.SecretDataMap) (string, error) {
	password := strings.TrimSpace(string(data[centralDBPasswordKey]))
	if password == "" || strings.ContainsAny(password, "\r\n") {
		return "", errors.Errorf("secret must contain a non-empty, single-line %q entry", centralDBPasswordKey)
	}
	return password, nil
}
