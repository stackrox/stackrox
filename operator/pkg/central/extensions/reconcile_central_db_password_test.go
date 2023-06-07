package extensions

import (
	"context"
	"testing"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileDBPassword(t *testing.T) {
	const (
		customPWSecretName = "my-password"
	)

	specWithoutSecretReference := v1alpha1.CentralSpec{
		Central: &v1alpha1.CentralComponentSpec{
			DB: &v1alpha1.CentralDBSpec{},
		},
	}

	specWithSecretReference := func(secretName string) v1alpha1.CentralSpec {
		return v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				DB: &v1alpha1.CentralDBSpec{
					PasswordSecret: &v1alpha1.LocalSecretReference{
						Name: secretName,
					},
				},
			},
		}
	}

	customPasswordSecret := func(secretName, password string) *v1.Secret {
		return secretWithValues(secretName, centralDBPasswordKey, password)
	}
	emptySecret := func(secretName string) *v1.Secret {
		return secretWithValues(secretName)
	}
	centralDBSecretWithPassword := func(password string) *v1.Secret {
		return secretWithValues(canonicalCentralDBPasswordSecretName, centralDBPasswordKey, password)
	}
	emptyCentralDbSecret := func() *v1.Secret {
		return emptySecret(canonicalCentralDBPasswordSecretName)
	}

	var cli ctrlClient.WithWatch
	var ctx context.Context
	setup := func() {
		ctx = context.Background()
		sch := runtime.NewScheme()
		require.NoError(t, platform.AddToScheme(sch))
		require.NoError(t, scheme.AddToScheme(sch))
		cli = fake.NewClientBuilder().WithScheme(sch).Build()
	}

	createCentral := func(ctx context.Context, spec v1alpha1.CentralSpec) (*v1alpha1.Central, error) {
		central := &v1alpha1.Central{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "platform.stackrox.io/v1alpha1",
				Kind:       "Central",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "central",
				Namespace: testutils.TestNamespace,
				UID:       "1234",
			},
			Spec: spec,
		}
		return central, cli.Create(ctx, central)
	}

	getCentralDBSecret := func(ctx context.Context) (*v1.Secret, error) {
		secret := &v1.Secret{}
		err := cli.Get(ctx, ctrlClient.ObjectKey{Name: canonicalCentralDBPasswordSecretName, Namespace: testutils.TestNamespace}, secret)
		return secret, err
	}

	assertDBSecretValue := func(t *testing.T, ctx context.Context, expectedPassword string) {
		secret, err := getCentralDBSecret(ctx)
		require.NoError(t, err)
		assert.Equal(t, expectedPassword, string(secret.Data[centralDBPasswordKey]))
	}

	t.Run("When the central-db-password secret exist with a Central owner reference, the owner reference should be removed", func(t *testing.T) {
		// ROX-13947: If we delete the secret when Centrals are deleted, but the PVC is not deleted, then Central
		// will no longer be able to connect to the DB upon reinstall. This removes any previous owner reference
		// that would've caused garbage collection to delete the secret.
		setup()
		central, err := createCentral(ctx, specWithoutSecretReference)
		require.NoError(t, err)
		secretWithOwner := centralDBSecretWithPassword("password")
		secretWithOwner.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "platform.stackrox.io/v1alpha1",
				Kind:       "Central",
				Name:       "central",
				UID:        "1234",
			},
		}
		err = cli.Create(ctx, secretWithOwner)
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		secret, err := getCentralDBSecret(ctx)
		require.NoError(t, err)
		assert.Empty(t, secret.OwnerReferences)
	})

	t.Run("When the central-db-password secret exist with a owner reference that is not Central, the owner reference should be untouched", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithoutSecretReference)
		require.NoError(t, err)
		secretWithOwner := centralDBSecretWithPassword("password")
		secretWithOwner.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "foo.com/v1alpha1",
				Kind:       "Central",
				Name:       "central",
				UID:        "1234",
			},
		}
		require.NoError(t, cli.Create(ctx, secretWithOwner))

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		secret, err := getCentralDBSecret(ctx)
		require.NoError(t, err)
		assert.Len(t, secret.OwnerReferences, 1)
	})

	t.Run("When no custom secret is specified, and no central-db-password secret exists, a password should be automatically generated", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithoutSecretReference)
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		secret, err := getCentralDBSecret(ctx)
		require.NoError(t, err)
		assert.NotEmptyf(t, secret.Data[centralDBPasswordKey], "Expected password to be generated")
	})

	t.Run("When no custom secret is specified, and a central-db-password secret exists without a password key, a password should be added", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithoutSecretReference)
		require.NoError(t, err)
		err = cli.Create(ctx, emptyCentralDbSecret())
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		secret, err := getCentralDBSecret(ctx)
		require.NoError(t, err)
		assert.NotEmptyf(t, secret.Data[centralDBPasswordKey], "Expected password to be generated")
	})

	t.Run("When no custom secret is specified, and a central-db-password secret exists, it should remain unchanged", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithoutSecretReference)
		require.NoError(t, err)
		err = cli.Create(ctx, centralDBSecretWithPassword("password"))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		assertDBSecretValue(t, ctx, "password")
	})

	t.Run("When no custom secret is specified, and the database is external, then an error should be thrown", func(t *testing.T) {
		setup()
		var someConnectionString = "bla"
		central, err := createCentral(ctx, v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				DB: &v1alpha1.CentralDBSpec{
					ConnectionStringOverride: &someConnectionString,
				},
			},
		})
		require.NoError(t, err)
		err = cli.Create(ctx, centralDBSecretWithPassword("password"))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "spec.central.db.passwordSecret must be set when using an external database")
	})

	t.Run("When a custom secret is specified, and no central-db-password secret exists, a central-db-password secret should be created with the value of the custom secret", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(customPWSecretName))
		require.NoError(t, err)
		err = cli.Create(ctx, customPasswordSecret(customPWSecretName, "password"))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		assertDBSecretValue(t, ctx, "password")
	})

	t.Run("When a custom secret is specified, and a central-db-password secret exists, the value should be changed to reflect the custom secret", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(customPWSecretName))
		require.NoError(t, err)
		err = cli.Create(ctx, customPasswordSecret(customPWSecretName, "password"))
		require.NoError(t, err)
		err = cli.Create(ctx, centralDBSecretWithPassword("old-password"))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		assertDBSecretValue(t, ctx, "password")
	})

	t.Run("When a custom secret is specified but doesn't exist, an error should be returned", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(customPWSecretName))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to get spec.central.db.passwordSecret")
	})
	t.Run("When a custom secret is specified but without a password key, an error should be returned", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(customPWSecretName))
		require.NoError(t, err)
		err = cli.Create(ctx, emptySecret(customPWSecretName))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "secret \"my-password\" does not contain a \"password\" entry")
	})
	t.Run("When a custom secret value changes, it should be reflected in the generated secret", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(customPWSecretName))
		require.NoError(t, err)

		// create the secret
		err = cli.Create(ctx, secretWithValues(customPWSecretName, "password", "password1"))
		require.NoError(t, err)

		// reconcile
		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		// check the generated secret has the initial value
		assertDBSecretValue(t, ctx, "password1")

		// update the secret
		err = cli.Update(ctx, secretWithValues(customPWSecretName, "password", "password2"))
		require.NoError(t, err)

		// reconcile
		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		// check the generated secret has been updated
		assertDBSecretValue(t, ctx, "password2")
	})

	t.Run("When a custom secret is specified but the password is invalid, an error should be returned", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(customPWSecretName))
		require.NoError(t, err)
		err = cli.Create(ctx, customPasswordSecret(customPWSecretName, " "))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "secret \"my-password\" contains an empty \"password\" entry")
	})
	t.Run("When a custom secret name = central-db-password but doesn't exist, an error should be returned", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(canonicalCentralDBPasswordSecretName))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "failed to get spec.central.db.passwordSecret")
	})

	t.Run("When the custom secret name = central-db-password one but has an invalid password, an error should be returned", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(canonicalCentralDBPasswordSecretName))
		require.NoError(t, err)
		err = cli.Create(ctx, customPasswordSecret(canonicalCentralDBPasswordSecretName, " "))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "secret \"central-db-password\" contains an empty \"password\" entry")
	})

	t.Run("When the custom secret name = central-db-password and is valid, it should be left untouched", func(t *testing.T) {
		setup()
		central, err := createCentral(ctx, specWithSecretReference(canonicalCentralDBPasswordSecretName))
		require.NoError(t, err)
		err = cli.Create(ctx, customPasswordSecret(canonicalCentralDBPasswordSecretName, "password"))
		require.NoError(t, err)

		err = reconcileCentralDBPassword(ctx, central, cli)
		require.NoError(t, err)

		assertDBSecretValue(t, ctx, "password")
	})

}

func secretWithValues(secretName string, keyValues ...string) *v1.Secret {
	var data = make(map[string][]byte)
	for i := 0; i < len(keyValues); i += 2 {
		data[keyValues[i]] = []byte(keyValues[i+1])
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: testutils.TestNamespace,
		},
		Data: data,
	}
}
