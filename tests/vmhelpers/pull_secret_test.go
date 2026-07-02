//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func writeDockerConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/config.json"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestEnsureImagePullSecret_UpdatesExistingSecretUsingFetchedResourceVersion(t *testing.T) {
	t.Parallel()

	const namespace = "vm-scan-test"
	secretPath := writeDockerConfigFile(t, `{"auths":{"quay.io":{"auth":"new"}}}`)

	client := kubefake.NewSimpleClientset(
		&coreV1.Secret{
			ObjectMeta: metaV1.ObjectMeta{
				Name:            ImagePullSecretName,
				Namespace:       namespace,
				ResourceVersion: "7",
			},
			Type: coreV1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				coreV1.DockerConfigJsonKey: []byte(`{"auths":{"quay.io":{"auth":"old"}}}`),
			},
		},
		&coreV1.ServiceAccount{
			ObjectMeta: metaV1.ObjectMeta{
				Name:      "default",
				Namespace: namespace,
			},
		},
	)
	client.PrependReactor("update", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAction := action.(k8stesting.UpdateAction)
		secret := updateAction.GetObject().(*coreV1.Secret)
		if secret.ResourceVersion == "" {
			return true, nil, apierrors.NewInvalid(
				coreV1.SchemeGroupVersion.WithKind("Secret").GroupKind(),
				secret.Name,
				field.ErrorList{field.Required(field.NewPath("metadata", "resourceVersion"), "must be set for an update")},
			)
		}
		return false, nil, nil
	})

	err := EnsureImagePullSecret(t.Context(), client, t.Logf, namespace, ImagePullSecretName, secretPath)
	require.NoError(t, err)

	secret, err := client.CoreV1().Secrets(namespace).Get(t.Context(), ImagePullSecretName, metaV1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "7", secret.ResourceVersion)
	require.Equal(t, `{"auths":{"quay.io":{"auth":"new"}}}`, string(secret.Data[coreV1.DockerConfigJsonKey]))

	sa, err := client.CoreV1().ServiceAccounts(namespace).Get(t.Context(), "default", metaV1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: ImagePullSecretName})
}

func TestEnsureImagePullSecret_WaitsForDefaultServiceAccountToAppear(t *testing.T) {
	t.Parallel()

	const namespace = "vm-scan-test"
	secretPath := writeDockerConfigFile(t, `{"auths":{"quay.io":{"auth":"new"}}}`)
	client := kubefake.NewSimpleClientset()

	var getAttempts atomic.Int32
	client.PrependReactor("get", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		getAction := action.(k8stesting.GetAction)
		if getAction.GetName() != "default" || getAction.GetNamespace() != namespace {
			return false, nil, nil
		}

		if getAttempts.Add(1) == 1 {
			require.NoError(t, client.Tracker().Add(&coreV1.ServiceAccount{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "default",
					Namespace: namespace,
				},
			}))
			return true, nil, apierrors.NewNotFound(coreV1.Resource("serviceaccounts"), "default")
		}
		return false, nil, nil
	})

	err := EnsureImagePullSecret(t.Context(), client, t.Logf, namespace, ImagePullSecretName, secretPath)
	require.NoError(t, err)

	require.GreaterOrEqual(t, getAttempts.Load(), int32(2))
	sa, err := client.CoreV1().ServiceAccounts(namespace).Get(t.Context(), "default", metaV1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: ImagePullSecretName})
}

func TestEnsureImagePullSecret_RetriesServiceAccountLinkOnConflict(t *testing.T) {
	t.Parallel()

	const namespace = "vm-scan-test"
	secretPath := writeDockerConfigFile(t, `{"auths":{"quay.io":{"auth":"new"}}}`)

	client := kubefake.NewSimpleClientset(
		&coreV1.ServiceAccount{
			ObjectMeta: metaV1.ObjectMeta{Name: "default", Namespace: namespace},
		},
	)
	updateAttempts := 0
	client.PrependReactor("update", "serviceaccounts", func(action k8stesting.Action) (bool, runtime.Object, error) {
		updateAttempts++
		if updateAttempts == 1 {
			return true, nil, apierrors.NewConflict(
				schema.GroupResource{Resource: "serviceaccounts"},
				"default",
				errors.New("conflict"),
			)
		}
		return false, nil, nil
	})

	err := EnsureImagePullSecret(t.Context(), client, t.Logf, namespace, ImagePullSecretName, secretPath)
	require.NoError(t, err)

	require.GreaterOrEqual(t, updateAttempts, 2)
	sa, err := client.CoreV1().ServiceAccounts(namespace).Get(t.Context(), "default", metaV1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, sa.ImagePullSecrets, coreV1.LocalObjectReference{Name: ImagePullSecretName})
}
