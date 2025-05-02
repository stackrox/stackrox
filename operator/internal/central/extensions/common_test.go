package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

type secretVerifyFunc func(t *testing.T, data types.SecretDataMap)
type statusVerifyFunc func(t *testing.T, status *platform.CentralStatus)

type secretReconciliationTestCase struct {
	Spec                   platform.CentralSpec
	Deleted                bool
	Existing               []*v1.Secret
	ExistingManaged        []*v1.Secret
	Other                  []ctrlClient.Object
	InterceptedK8sAPICalls interceptor.Funcs

	ExpectedCreatedSecrets     map[string]secretVerifyFunc
	ExpectedError              string
	ExpectedNotExistingSecrets []string
	VerifyStatus               statusVerifyFunc
}

func basicSpecWithScanner(scannerEnabled bool, scannerV4Enabled bool) platform.CentralSpec {
	spec := platform.CentralSpec{
		Scanner: &platform.ScannerComponentSpec{
			ScannerComponent: new(platform.ScannerComponentPolicy),
		},
		ScannerV4: &platform.ScannerV4Spec{
			ScannerComponent: new(platform.ScannerV4ComponentPolicy),
		},
	}
	if scannerEnabled {
		*spec.Scanner.ScannerComponent = platform.ScannerComponentEnabled
	} else {
		*spec.Scanner.ScannerComponent = platform.ScannerComponentDisabled
	}

	if scannerV4Enabled {
		*spec.ScannerV4.ScannerComponent = platform.ScannerV4ComponentEnabled
	} else {
		*spec.ScannerV4.ScannerComponent = platform.ScannerV4ComponentDisabled
	}

	return spec
}

func testSecretReconciliation(t *testing.T, runFn func(ctx context.Context, central *platform.Central, client ctrlClient.Client, direct ctrlClient.Reader, statusUpdater func(updateStatusFunc), log logr.Logger) error, c secretReconciliationTestCase) {
	central := buildFakeCentral(c)
	client := buildFakeClient(t, c, central)
	statusUpdater := func(statusFunc updateStatusFunc) {
		statusFunc(&central.Status)
	}

	// Verify that an initial invocation does not touch any of the existing unmanaged secrets, and creates
	// the expected managed ones.
	err := runFn(context.Background(), central.DeepCopy(), client, client, statusUpdater, logr.Discard())
	if c.ExpectedError == "" {
		require.NoError(t, err)
	} else {
		require.Error(t, err)
		assert.Contains(t, err.Error(), c.ExpectedError)
		return
	}

	if c.VerifyStatus != nil {
		c.VerifyStatus(t, &central.Status)
	}

	secretsList, secretsByName := listSecrets(t, client)
	verifyUnmanagedSecretsNotChanged(t, c.Existing, secretsByName)
	verifyNotCreatedSecrets(t, c.ExpectedNotExistingSecrets, secretsByName)
	verifyCreatedSecrets(t, c.ExpectedCreatedSecrets, secretsByName, central.Name)
	assert.Empty(t, secretsByName, "one or more unexpected secrets exist")

	// Verify that a second invocation does not further change the cluster state
	err = runFn(context.Background(), central.DeepCopy(), client, client, statusUpdater, logr.Discard())
	assert.NoError(t, err, "second invocation of reconciliation function failed")
	if c.VerifyStatus != nil {
		c.VerifyStatus(t, &central.Status)
	}
	verifySecretsMatch(t, client, secretsList)

	// Fake deletion of the CR
	central.DeletionTimestamp = new(metav1.Time)
	*central.DeletionTimestamp = metav1.Now()

	err = runFn(context.Background(), central.DeepCopy(), client, client, statusUpdater, logr.Discard())
	assert.NoError(t, err, "deletion of CR resulted in error")

	_, postDeletionSecretsByName := listSecrets(t, client)
	verifyUnmanagedSecretsNotChanged(t, c.Existing, postDeletionSecretsByName)
	assert.Empty(t, postDeletionSecretsByName, "newly created secrets remain after deletion")
}

func buildFakeCentral(c secretReconciliationTestCase) *platform.Central {
	central := &platform.Central{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "Central",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-central",
			Namespace: testutils.TestNamespace,
		},
		Spec: *c.Spec.DeepCopy(),
	}

	if c.Deleted {
		central.DeletionTimestamp = new(metav1.Time)
		*central.DeletionTimestamp = metav1.Now()
	}

	return central
}

func buildFakeClient(t *testing.T, c secretReconciliationTestCase, central *platform.Central) ctrlClient.Client {
	var existingSecrets []ctrlClient.Object
	for _, existingSecret := range c.Existing {
		existingSecrets = append(existingSecrets, existingSecret.DeepCopy())
	}
	for _, existingManagedSecret := range c.ExistingManaged {
		managedSecret := existingManagedSecret.DeepCopy()
		managedSecret.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(central, central.GroupVersionKind())})
		existingSecrets = append(existingSecrets, managedSecret)
	}
	var otherExisting []runtime.Object
	for _, existingObj := range c.Other {
		otherExisting = append(otherExisting, existingObj.DeepCopyObject())
	}

	sch := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(sch))
	require.NoError(t, scheme.AddToScheme(sch))
	client := fake.NewClientBuilder().
		WithScheme(sch).
		WithObjects(existingSecrets...).
		WithRuntimeObjects(otherExisting...).
		Build()

	return interceptor.NewClient(client, c.InterceptedK8sAPICalls)
}

func listSecrets(t *testing.T, client ctrlClient.Client) (*v1.SecretList, map[string]v1.Secret) {
	secretsList := &v1.SecretList{}
	err := client.List(context.Background(), secretsList, ctrlClient.InNamespace(testutils.TestNamespace))
	require.NoError(t, err)

	secretsByName := make(map[string]v1.Secret)
	for _, secret := range secretsList.Items {
		secretsByName[secret.Name] = secret
	}
	return secretsList, secretsByName
}

func verifyUnmanagedSecretsNotChanged(t *testing.T, existing []*v1.Secret, secretsByName map[string]v1.Secret) {
	for _, existingSecret := range existing {
		found, ok := secretsByName[existingSecret.Name]
		if !assert.Truef(t, ok, "pre-existing unmanaged secret %s is gone", existingSecret.Name) {
			continue
		}
		assert.Equalf(t, existingSecret.Data, found.Data, "data of pre-existing unmanaged secret %s has changed", existingSecret.Name)
		delete(secretsByName, existingSecret.Name)
	}
}

func verifyNotCreatedSecrets(t *testing.T, expectedNotExisting []string, secretsByName map[string]v1.Secret) {
	for _, name := range expectedNotExisting {
		_, ok := secretsByName[name]
		assert.Falsef(t, ok, "secret %s was created", name)
	}
}

func verifyCreatedSecrets(
	t *testing.T,
	expectedCreatedSecrets map[string]secretVerifyFunc,
	secretsByName map[string]v1.Secret,
	ownerName string,
) {
	for name, verifyFunc := range expectedCreatedSecrets {
		found, ok := secretsByName[name]
		if !assert.True(t, ok, "expected secret %s was not created", name) {
			continue
		}
		hasOwnerRef := false
		for _, ownerRef := range found.ObjectMeta.GetOwnerReferences() {
			if ownerRef.Name == ownerName {
				hasOwnerRef = true
			}
		}
		assert.Truef(t, hasOwnerRef, "newly created secret %s is missing owner reference", name)
		verifyFunc(t, found.Data)
		delete(secretsByName, name)
	}
}

func verifySecretsMatch(
	t *testing.T,
	client ctrlClient.Client,
	initialSecrets *v1.SecretList,
) {
	secretsList := &v1.SecretList{}
	err := client.List(context.Background(), secretsList, ctrlClient.InNamespace(testutils.TestNamespace))
	require.NoError(t, err)

	assert.ElementsMatch(t, initialSecrets.Items, secretsList.Items, "second invocation changed the cluster state")
}
