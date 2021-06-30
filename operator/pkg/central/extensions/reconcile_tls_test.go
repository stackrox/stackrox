package extensions

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/logr"
	centralv1Alpha1 "github.com/stackrox/rox/operator/api/central/v1alpha1"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type secretVerifyFunc func(t *testing.T, data dataMap)

func verifyCentralCert(t *testing.T, data dataMap) {
	ca, err := certgen.LoadCAFromFileMap(data)
	require.NoError(t, err)
	assert.NoError(t, certgen.VerifyServiceCert(data, ca, mtls.CentralSubject, ""))

	_, err = certgen.LoadJWTSigningKeyFromFileMap(data)
	assert.NoError(t, err)
}

func verifyServiceCert(subj mtls.Subject) secretVerifyFunc {
	return func(t *testing.T, data dataMap) {
		validatingCA, err := mtls.LoadCAForValidation(data["ca.pem"])
		require.NoError(t, err)

		assert.NoError(t, certgen.VerifyServiceCert(data, validatingCA, subj, ""))
	}
}

func TestCreateCentralTLS(t *testing.T) {
	testCA, err := certgen.GenerateCA()
	require.NoError(t, err)

	centralFileMap := make(dataMap)
	certgen.AddCAToFileMap(centralFileMap, testCA)
	require.NoError(t, certgen.IssueCentralCert(centralFileMap, testCA))
	jwtKey, err := certgen.GenerateJWTSigningKey()
	require.NoError(t, err)
	certgen.AddJWTSigningKeyToFileMap(centralFileMap, jwtKey)

	existingCentral := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: "testns",
		},
		Data: centralFileMap,
	}

	scannerFileMap := make(dataMap)
	certgen.AddCACertToFileMap(scannerFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerSubject, ""))

	existingScanner := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-tls",
			Namespace: "testns",
		},
		Data: scannerFileMap,
	}

	scannerDBFileMap := make(dataMap)
	certgen.AddCACertToFileMap(scannerDBFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerDBFileMap, testCA, mtls.ScannerDBSubject, ""))

	existingScannerDB := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-tls",
			Namespace: "testns",
		},
		Data: scannerDBFileMap,
	}

	cases := map[string]struct {
		ScannerEnabled bool
		Deleted        bool
		Existing       []*v1.Secret

		ExpectedOwned map[string]secretVerifyFunc
		ExpectedError string
	}{
		"empty-state-no-scanner": {
			ScannerEnabled: false,
			ExpectedOwned: map[string]secretVerifyFunc{
				"central-tls": verifyCentralCert,
			},
		},
		"empty-state-with-scanner": {
			ScannerEnabled: true,
			ExpectedOwned: map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"scanner-tls":    verifyServiceCert(mtls.ScannerSubject),
				"scanner-db-tls": verifyServiceCert(mtls.ScannerDBSubject),
			},
		},
		"existing-central-no-scanner": {
			ScannerEnabled: false,
			Existing:       []*v1.Secret{existingCentral},
		},
		"existing-central-with-scanner": {
			ScannerEnabled: true,
			Existing:       []*v1.Secret{existingCentral},
			ExpectedOwned: map[string]secretVerifyFunc{
				"scanner-tls":    verifyServiceCert(mtls.ScannerSubject),
				"scanner-db-tls": verifyServiceCert(mtls.ScannerDBSubject),
			},
		},
		"all-existing-no-scanner": {
			ScannerEnabled: false,
			Existing:       []*v1.Secret{existingCentral, existingScanner, existingScannerDB},
		},
		"all-existing-with-scanner": {
			ScannerEnabled: true,
			Existing:       []*v1.Secret{existingCentral, existingScanner, existingScannerDB},
		},
		// TODO(ROX-7416): Test error cases
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			central := &centralv1Alpha1.Central{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "platform.stackrox.io/v1alpha1",
					Kind:       "Central",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-central",
					Namespace: "testns",
				},
				Spec: centralv1Alpha1.CentralSpec{
					Scanner: &centralv1Alpha1.ScannerComponentSpec{
						ScannerComponent: new(centralv1Alpha1.ScannerComponentPolicy),
					},
				},
			}

			if c.ScannerEnabled {
				*central.Spec.Scanner.ScannerComponent = centralv1Alpha1.ScannerComponentEnabled
			} else {
				*central.Spec.Scanner.ScannerComponent = centralv1Alpha1.ScannerComponentDisabled
			}

			u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(central)
			require.NoError(t, err)

			var allExisting []runtime.Object
			for _, existingSecret := range c.Existing {
				allExisting = append(allExisting, existingSecret)
			}

			client := fake.NewSimpleClientset(allExisting...)

			// Verify that an initial invocation does not touch any of the existing secrets, and creates
			// the expected ones.
			err = reconcileCentralTLS(context.Background(), &unstructured.Unstructured{Object: u}, client, logr.Discard())
			if c.ExpectedError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.ExpectedError)
				return
			}

			secretsList, err := client.CoreV1().Secrets("testns").List(context.Background(), metav1.ListOptions{})
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

			for name, verifyFunc := range c.ExpectedOwned {
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
			err = reconcileCentralTLS(context.Background(), &unstructured.Unstructured{Object: u}, client, logr.Discard())
			assert.NoError(t, err, "second invocation of reconcileCentralTLS failed")

			secretsList2, err := client.CoreV1().Secrets("testns").List(context.Background(), metav1.ListOptions{})
			require.NoError(t, err)

			assert.ElementsMatch(t, secretsList.Items, secretsList2.Items, "second invocation changed the cluster state")

			// Fake deletion of the CR
			central.DeletionTimestamp = new(metav1.Time)
			*central.DeletionTimestamp = metav1.NewTime(time.Now())

			u, err = runtime.DefaultUnstructuredConverter.ToUnstructured(central)
			require.NoError(t, err)

			err = reconcileCentralTLS(context.Background(), &unstructured.Unstructured{Object: u}, client, logr.Discard())
			assert.NoError(t, err, "deletion of CR resulted in error")

			secretsList3, err := client.CoreV1().Secrets("testns").List(context.Background(), metav1.ListOptions{})
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
		})
	}
}
