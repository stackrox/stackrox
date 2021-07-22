package extensions

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	centralv1Alpha1 "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testNamespace = `testns`
)

type secretVerifyFunc func(t *testing.T, data secretDataMap)
type statusVerifyFunc func(t *testing.T, status *centralv1Alpha1.CentralStatus)

type secretReconciliationTestCase struct {
	Spec     centralv1Alpha1.CentralSpec
	Deleted  bool
	Existing []*v1.Secret

	ExpectedCreatedSecrets map[string]secretVerifyFunc
	ExpectedError          string
	VerifyStatus           statusVerifyFunc
}

func basicSpecWithScanner(scannerEnabled bool) centralv1Alpha1.CentralSpec {
	spec := centralv1Alpha1.CentralSpec{
		Scanner: &centralv1Alpha1.ScannerComponentSpec{
			ScannerComponent: new(centralv1Alpha1.ScannerComponentPolicy),
		},
	}
	if scannerEnabled {
		*spec.Scanner.ScannerComponent = centralv1Alpha1.ScannerComponentEnabled
	} else {
		*spec.Scanner.ScannerComponent = centralv1Alpha1.ScannerComponentDisabled
	}
	return spec
}

func testSecretReconciliation(t *testing.T, runFn func(ctx context.Context, central *centralv1Alpha1.Central, k8sClient kubernetes.Interface, statusUpdater func(updateStatusFunc), log logr.Logger) error, c secretReconciliationTestCase) {
	central := &centralv1Alpha1.Central{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "Central",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-central",
			Namespace: testNamespace,
		},
		Spec: *c.Spec.DeepCopy(),
	}

	if c.Deleted {
		central.DeletionTimestamp = new(metav1.Time)
		*central.DeletionTimestamp = metav1.Now()
	}

	statusUpdater := func(statusFunc updateStatusFunc) {
		statusFunc(&central.Status)
	}

	var allExisting []runtime.Object
	for _, existingSecret := range c.Existing {
		allExisting = append(allExisting, existingSecret.DeepCopy())
	}

	client := fake.NewSimpleClientset(allExisting...)

	// Verify that an initial invocation does not touch any of the existing secrets, and creates
	// the expected ones.
	err := runFn(context.Background(), central.DeepCopy(), client, statusUpdater, logr.Discard())
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

	secretsList, err := client.CoreV1().Secrets(testNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)

	secretsByName := make(map[string]v1.Secret)
	for _, secret := range secretsList.Items {
		secretsByName[secret.Name] = secret
	}

	for _, existingSecret := range c.Existing {
		found, ok := secretsByName[existingSecret.Name]
		if !assert.Truef(t, ok, "pre-existing secret %s is gone", existingSecret.Name) {
			continue
		}
		assert.Equalf(t, existingSecret.Data, found.Data, "data of pre-existing secret %s has changed", existingSecret.Name)
		delete(secretsByName, existingSecret.Name)
	}

	for name, verifyFunc := range c.ExpectedCreatedSecrets {
		found, ok := secretsByName[name]
		if !assert.True(t, ok, "expected secret %s was not created", name) {
			continue
		}
		hasOwnerRef := false
		for _, ownerRef := range found.ObjectMeta.GetOwnerReferences() {
			if ownerRef.Name == "test-central" {
				hasOwnerRef = true
			}
		}
		assert.Truef(t, hasOwnerRef, "newly created secret %s is missing owner reference", name)
		verifyFunc(t, found.Data)
		delete(secretsByName, name)
	}

	assert.Empty(t, secretsByName, "one or more unexpected secrets exist")

	// Verify that a second invocation does not further change the cluster state
	err = runFn(context.Background(), central.DeepCopy(), client, statusUpdater, logr.Discard())
	assert.NoError(t, err, "second invocation of reconciliation function failed")

	if c.VerifyStatus != nil {
		c.VerifyStatus(t, &central.Status)
	}

	secretsList2, err := client.CoreV1().Secrets(testNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)

	assert.ElementsMatch(t, secretsList.Items, secretsList2.Items, "second invocation changed the cluster state")

	// Fake deletion of the CR
	central.DeletionTimestamp = new(metav1.Time)
	*central.DeletionTimestamp = metav1.Now()

	err = runFn(context.Background(), central.DeepCopy(), client, statusUpdater, logr.Discard())
	assert.NoError(t, err, "deletion of CR resulted in error")

	secretsList3, err := client.CoreV1().Secrets(testNamespace).List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)

	postDeletionSecretsByName := make(map[string]v1.Secret)
	for _, secret := range secretsList3.Items {
		postDeletionSecretsByName[secret.Name] = secret
	}

	// Verify pre-existing secrets still exist
	for _, existingSecret := range c.Existing {
		found, ok := postDeletionSecretsByName[existingSecret.Name]
		if !assert.Truef(t, ok, "pre-existing secret %s is gone", existingSecret.Name) {
			continue
		}
		assert.Equalf(t, existingSecret.Data, found.Data, "data of pre-existing secret %s has changed", existingSecret.Name)
		delete(postDeletionSecretsByName, existingSecret.Name)
	}

	assert.Empty(t, postDeletionSecretsByName, "newly created secrets remain after deletion")
}
