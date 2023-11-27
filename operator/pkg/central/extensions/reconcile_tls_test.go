package extensions

import (
	"context"
	"crypto/x509"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func verifyCentralCert(t *testing.T, data types.SecretDataMap) {
	ca, err := certgen.LoadCAFromFileMap(data)
	require.NoError(t, err)
	assert.NoError(t, certgen.VerifyServiceCert(data, ca, storage.ServiceType_CENTRAL_SERVICE, ""))

	_, err = certgen.LoadJWTSigningKeyFromFileMap(data)
	assert.NoError(t, err)
}

func verifyCentralServiceCert(serviceType storage.ServiceType) secretVerifyFunc {
	return verifyServiceCert(serviceType, "")
}

func verifySecuredClusterServiceCert(serviceType storage.ServiceType) secretVerifyFunc {
	return verifyServiceCert(serviceType, services.ServiceTypeToSlugName(serviceType)+"-")
}

func verifyServiceCert(serviceType storage.ServiceType, fileNamePrefix string) secretVerifyFunc {
	return func(t *testing.T, data types.SecretDataMap) {
		validatingCA, err := mtls.LoadCAForValidation(data["ca.pem"])
		require.NoError(t, err)

		assert.NoError(t, certgen.VerifyServiceCert(data, validatingCA, serviceType, fileNamePrefix))
	}
}

func TestCreateCentralTLS(t *testing.T) {
	testCA, err := certgen.GenerateCA()
	require.NoError(t, err)

	centralFileMap := make(types.SecretDataMap)
	certgen.AddCAToFileMap(centralFileMap, testCA)
	require.NoError(t, certgen.IssueCentralCert(centralFileMap, testCA))
	jwtKey, err := certgen.GenerateJWTSigningKey()
	require.NoError(t, err)
	certgen.AddJWTSigningKeyToFileMap(centralFileMap, jwtKey)

	existingCentral := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: centralFileMap,
	}

	centralDBFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(centralDBFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(centralDBFileMap, testCA, mtls.CentralDBSubject, ""))

	existingCentralDB := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-db-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: centralDBFileMap,
	}

	scannerFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(scannerFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerSubject, ""))
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerV4IndexerSubject, ""))
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerV4MatcherSubject, ""))
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerV4DBSubject, ""))

	existingScanner := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: scannerFileMap,
	}

	scannerDBFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(scannerDBFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerDBFileMap, testCA, mtls.ScannerDBSubject, ""))

	existingScannerDB := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: scannerDBFileMap,
	}

	cases := map[string]secretReconciliationTestCase{
		"When no secrets exist and scanner is disabled, a managed central-tls and central-db-tls secrets should be created": {
			Spec: basicSpecWithScanner(false),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
			},
		},
		"When no secrets exist and scanner is disabled but secured cluster exists, a managed central-tls secret and init bundle secrets should be created": {
			Spec: basicSpecWithScanner(false),
			Other: []ctrlClient.Object{&platform.SecuredCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secured-cluster-services",
					Namespace: testutils.TestNamespace,
				},
			}},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":           verifyCentralCert,
				"central-db-tls":        verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
				"admission-control-tls": verifySecuredClusterServiceCert(storage.ServiceType_ADMISSION_CONTROL_SERVICE),
				"collector-tls":         verifySecuredClusterServiceCert(storage.ServiceType_COLLECTOR_SERVICE),
				"sensor-tls":            verifySecuredClusterServiceCert(storage.ServiceType_SENSOR_SERVICE),
			},
		},
		"When no secrets exist and scanner is enabled, all managed secrets should be created": {
			Spec: basicSpecWithScanner(true),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
				"scanner-tls":    verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
				"scanner-db-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
			},
		},
		"When a valid unmanaged central-tls and central-db-tls secrets exist and scanner is disabled, no further secrets should be created": {
			Spec:     basicSpecWithScanner(false),
			Existing: []*v1.Secret{existingCentral, existingCentralDB},
		},
		"When a valid unmanaged central-tls and central-db-tls secrets exist and scanner is enabled, managed secrets should be created for scanner": {
			Spec:     basicSpecWithScanner(true),
			Existing: []*v1.Secret{existingCentral, existingCentralDB},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"scanner-tls":    verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
				"scanner-db-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
			},
		},
		"When valid unmanaged secrets exist for everything and scanner is disabled, no secrets should be created or deleted": {
			Spec:     basicSpecWithScanner(false),
			Existing: []*v1.Secret{existingCentral, existingCentralDB, existingScanner, existingScannerDB},
		},
		"When valid unmanaged secrets exist for everything and scanner is enabled, no secrets should be created or deleted": {
			Spec:     basicSpecWithScanner(true),
			Existing: []*v1.Secret{existingCentral, existingCentralDB, existingScanner, existingScannerDB},
		},
		"When creating a new central-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false),
			InterceptedK8sAPICalls: creatingSecretFails("central-tls"),
			ExpectedError:          "reconciling central-tls secret",
		},
		"When creating a new central-db-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false),
			InterceptedK8sAPICalls: creatingSecretFails("central-db-tls"),
			ExpectedError:          "reconciling central-db-tls secret",
		},
		"When creating a new scanner-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(true),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-tls"),
			ExpectedError:          "reconciling scanner-tls secret",
		},
		"When creating a new scanner-db-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(true),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-db-tls"),
			ExpectedError:          "reconciling scanner-db-tls secret",
		},
		"When getting an existing central-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false),
			InterceptedK8sAPICalls: gettingSecretFails("central-tls"),
			ExpectedError:          "reconciling central-tls secret",
		},
		"When getting an existing central-db-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false),
			InterceptedK8sAPICalls: gettingSecretFails("central-db-tls"),
			ExpectedError:          "reconciling central-db-tls secret",
		},
		"When getting an existing scanner-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(true),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-tls"),
			ExpectedError:          "reconciling scanner-tls secret",
		},
		"When getting an existing scanner-db-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(true),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-db-tls"),
			ExpectedError:          "reconciling scanner-db-tls secret",
		},
		"When deleting an existing central-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingCentral},
			InterceptedK8sAPICalls: deletingSecretFails("central-tls"),
			ExpectedError:          "reconciling central-tls secret",
		},
		"When deleting an existing central-tls secret fails with a 404, an error should not be returned because the secret is likely to be already deleted": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingCentral},
			InterceptedK8sAPICalls: secretIsAlreadyDeleted("central-tls"),
		},
		"When deleting an existing central-db-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingCentralDB},
			InterceptedK8sAPICalls: deletingSecretFails("central-db-tls"),
			ExpectedError:          "reconciling central-db-tls secret",
		},
		"When deleting an existing central-db-tls secret fails with a 404, an error should not be returned because the secret is likely to be already deleted": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingCentralDB},
			InterceptedK8sAPICalls: secretIsAlreadyDeleted("central-db-tls"),
		},
		"When deleting an existing scanner-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScanner},
			InterceptedK8sAPICalls: deletingSecretFails("scanner-tls"),
			ExpectedError:          "reconciling scanner-tls secret",
		},
		"When deleting an existing scanner-tls secret fails with a 404, an error should not be returned because the secret is likely to be already deleted": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScanner},
			InterceptedK8sAPICalls: secretIsAlreadyDeleted("scanner-tls"),
		},
		"When deleting an existing scanner-db-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScannerDB},
			InterceptedK8sAPICalls: deletingSecretFails("scanner-db-tls"),
			ExpectedError:          "reconciling scanner-db-tls secret",
		},
		"When deleting an existing scanner-db-tls secret fails with a 404, an error should not be returned because the secret is likely to be already deleted": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScannerDB},
			InterceptedK8sAPICalls: secretIsAlreadyDeleted("scanner-db-tls"),
		},
	}

	for name, c := range cases {
		c := c
		if strings.Contains(name, "init bundle secrets should be created") {
			// See ROX-9967.
			// TODO(ROX-9969): Remove this exclusion after the init-bundle cert rotation stabilization.
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileCentralTLS, c)
		})
	}
}

func gettingSecretFails(secretName string) interceptor.Funcs {
	return interceptor.Funcs{
		Get: func(ctx context.Context, client ctrlClient.WithWatch, key ctrlClient.ObjectKey, obj ctrlClient.Object, opts ...ctrlClient.GetOption) error {
			if _, ok := obj.(*v1.Secret); ok && key.Name == secretName {
				return k8sErrors.NewServiceUnavailable("failure")
			}
			return client.Get(ctx, key, obj, opts...)
		},
	}
}

func creatingSecretFails(secretName string) interceptor.Funcs {
	return interceptor.Funcs{
		Create: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.CreateOption) error {
			if secret, ok := obj.(*v1.Secret); ok && secret.Name == secretName {
				return k8sErrors.NewServiceUnavailable("failure")
			}
			return client.Create(ctx, obj, opts...)
		},
	}
}

func deletingSecretFails(secretName string) interceptor.Funcs {
	return interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error {
			if secret, ok := obj.(*v1.Secret); ok && secret.Name == secretName {
				return k8sErrors.NewServiceUnavailable("failure")
			}
			return client.Delete(ctx, obj, opts...)
		},
	}
}
func secretIsAlreadyDeleted(secretName string) interceptor.Funcs {
	return interceptor.Funcs{
		Delete: func(ctx context.Context, client ctrlClient.WithWatch, obj ctrlClient.Object, opts ...ctrlClient.DeleteOption) error {
			// To simulate that the secret was already deleted, this intercepted
			// call will actually delete the secret by calling the underlying client,
			// but it will return a 404 error. Useful for testing the cases where
			// some object was already deleted.
			err := client.Delete(ctx, obj, opts...)
			if secret, ok := obj.(*v1.Secret); ok && secret.Name == secretName {
				return k8sErrors.NewNotFound(v1.SchemeGroupVersion.WithResource("secrets").GroupResource(), secretName)
			}
			return err
		},
	}
}

func TestRenewInitBundle(t *testing.T) {
	type renewInitBundleTestCase struct {
		now                string
		notBefore          string
		notAfter           string
		expectError        bool
		expectErrorMessage string
	}

	cases := map[string]renewInitBundleTestCase{
		"should NOT refresh init-bundle when the certificate remains valid": {
			now:         "2021-02-11T12:00:00.000Z",
			notBefore:   "2021-02-11T00:00:00.000Z",
			notAfter:    "2021-02-11T23:59:59.000Z",
			expectError: false,
		},
		"should refresh init-bundle when the certificate is already expired": {
			now:                "2021-02-11T12:00:00.000Z",
			notBefore:          "2021-02-11T00:00:00.000Z",
			notAfter:           "2021-02-11T11:00:00.000Z",
			expectErrorMessage: "init bundle secret requires update, certificate is expired (or going to expire soon), not after: 2021-02-11 11:00:00 +0000 UTC, renew threshold: 2021-02-11 09:30:00 +0000 UTC",
		},
		"should refresh init-bundle when the certificate lifetime is not started": {
			now:                "2021-02-11T12:00:00.000Z",
			notBefore:          "2021-02-11T22:00:00.000Z",
			notAfter:           "2021-02-11T23:59:59.000Z",
			expectErrorMessage: "init bundle secret requires update, certificate lifetime starts in the future, not before: 2021-02-11 22:00:00 +0000 UTC",
		},
		"should refresh init-bundle when the certificate expires within the reconciliation period": {
			now:                "2021-02-11T12:00:00.000Z",
			notBefore:          "2021-02-11T00:00:00.000Z",
			notAfter:           "2021-02-11T12:30:00.000Z",
			expectErrorMessage: "init bundle secret requires update, certificate is expired (or going to expire soon), not after: 2021-02-11 12:30:00 +0000 UTC, renew threshold: 2021-02-11 11:00:00 +0000 UTC",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			notBefore, err := time.Parse(time.RFC3339, c.notBefore)
			require.NoError(t, err)
			notAfter, err := time.Parse(time.RFC3339, c.notAfter)
			require.NoError(t, err)
			cert := &x509.Certificate{
				NotBefore: notBefore,
				NotAfter:  notAfter,
			}
			now, err := time.Parse(time.RFC3339, c.now)
			require.NoError(t, err)

			if c.expectErrorMessage != "" {
				c.expectError = true
			}

			if c.expectError {
				if c.expectErrorMessage != "" {
					assert.EqualError(t, checkInitBundleCertRenewal(cert, now), c.expectErrorMessage)
				} else {
					assert.Error(t, checkInitBundleCertRenewal(cert, now))
				}
			} else {
				assert.NoError(t, checkInitBundleCertRenewal(cert, now))
			}
		})
	}
}

func Test_createCentralTLSExtensionRun_validateAndConsumeCentralTLSData(t *testing.T) {

	type testCase struct {
		fileMap types.SecretDataMap
		assert  func(t *testing.T, err error)
	}

	randomCA := func() (mtls.CA, error) {
		serial, err := mtls.RandomSerial()
		if err != nil {
			return nil, errors.New("could not generate serial number")
		}
		req := csr.CertificateRequest{
			CN:           "SomeCommonNameThatIsNotACS",
			KeyRequest:   csr.NewKeyRequest(),
			SerialNumber: serial.String(),
		}
		caCert, _, caKey, err := initca.New(&req)
		if err != nil {
			return nil, errors.New("could not generate CA")
		}
		return mtls.LoadCAForSigning(caCert, caKey)
	}

	ca1, err := certgen.GenerateCA()
	require.NoError(t, err)

	centralCertFromCA1, err := ca1.IssueCertForSubject(mtls.CentralSubject)
	require.NoError(t, err)

	unexpectedSubjectCertFromCA1, err := ca1.IssueCertForSubject(mtls.AdmissionControlSubject)
	require.NoError(t, err)

	ca2, err := certgen.GenerateCA()
	require.NoError(t, err)

	centralCertFromCA2, err := ca2.IssueCertForSubject(mtls.CentralSubject)
	require.NoError(t, err)

	caNotFromACS, err := randomCA()
	require.NoError(t, err)

	cases := map[string]testCase{
		"should fail if the CA was not issued by ACS": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName: caNotFromACS.CertPEM(),
				mtls.CAKeyFileName:  caNotFromACS.KeyPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "invalid certificate common name")
			},
		},
		"should fail if the CA is not... a CA": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName: centralCertFromCA1.CertPEM,
				mtls.CAKeyFileName:  centralCertFromCA1.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "certificate is not valid as CA")
			},
		},
		"should fail when the ca cert and key do not match": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName: ca1.CertPEM(),
				mtls.CAKeyFileName:  ca2.KeyPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "private key does not match public key")
			},
		},
		"should fail when the ca cert is missing": {
			fileMap: types.SecretDataMap{
				mtls.CAKeyFileName: ca1.KeyPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "no CA certificate in file map")
			},
		},
		"should fail when the ca key is missing": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName: ca1.CertPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "no CA key in file map")
			},
		},
		"should fail when the ca cert is invalid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName: []byte("invalid"),
				mtls.CAKeyFileName:  ca1.KeyPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to find any PEM data in certificate input")
			},
		},
		"should fail when the ca key is invalid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName: ca1.CertPEM(),
				mtls.CAKeyFileName:  []byte("invalid"),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to find any PEM data in key input")
			},
		},
		"should fail when the service cert is missing": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:     ca1.CertPEM(),
				mtls.CAKeyFileName:      ca1.KeyPEM(),
				mtls.ServiceKeyFileName: centralCertFromCA1.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "no service certificate in file map")
			},
		},
		"should fail when the service key is missing": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: centralCertFromCA1.CertPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "no service private key in file map")
			},
		},
		"should fail when the service cert does not match the service key": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:  centralCertFromCA2.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "mismatched certificate and private key")
			},
		},
		"should fail when the service cert is invalid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: []byte("invalid"),
				mtls.ServiceKeyFileName:  centralCertFromCA1.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "unparseable certificate in file map")
			},
		},
		"should fail when the service key is invalid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:  []byte("invalid"),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "invalid private key")
			},
		},
		"should fail when the service cert is not signed by the ca cert": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: centralCertFromCA2.CertPEM,
				mtls.ServiceKeyFileName:  centralCertFromCA2.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "certificate signed by unknown authority")
			},
		},
		"should fail when the service cert subject is not the expected service name": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: unexpectedSubjectCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:  unexpectedSubjectCertFromCA1.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "unexpected certificate service type")
			},
		},
		"should succeed when the ca cert and key are valid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:  centralCertFromCA1.KeyPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		// TODO(ROX-16206) Discuss tests around ca/service cert expiration, as well as "NotBefore".
		// Currently these verifications are only done for the init bundle secret reconciliation,
		// which has been disabled anyways. We also currently ignore CRLs and OCSPs.
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			r := &createCentralTLSExtensionRun{}
			err := r.validateAndConsumeCentralTLSData(tt.fileMap, true)
			tt.assert(t, err)
		})
	}
}

func Test_createCentralTLSExtensionRun_validateServiceTLSData(t *testing.T) {
	type testCase struct {
		ca      mtls.CA
		fileMap types.SecretDataMap
		assert  func(t *testing.T, err error)
	}

	subjects := []mtls.Subject{
		mtls.ScannerSubject,
		mtls.ScannerDBSubject,
		mtls.CentralDBSubject,
		mtls.ScannerV4IndexerSubject,
		mtls.ScannerV4MatcherSubject,
		mtls.ScannerV4DBSubject,
	}

	ca1, err := certgen.GenerateCA()
	require.NoError(t, err)

	unexpectedSubjectCertFromCA1, err := ca1.IssueCertForSubject(mtls.AdmissionControlSubject)
	require.NoError(t, err)

	ca2, err := certgen.GenerateCA()
	require.NoError(t, err)

	for _, subject := range subjects {
		t.Run(subject.Identifier, func(t *testing.T) {

			certForServiceFromCA1, err := ca1.IssueCertForSubject(subject)
			require.NoError(t, err)

			certForServiceFromCA2, err := ca2.IssueCertForSubject(subject)
			require.NoError(t, err)

			cases := map[string]testCase{
				"should fail when the certificate does not match the ca": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceKeyFileName:  certForServiceFromCA2.KeyPEM,
						mtls.ServiceCertFileName: certForServiceFromCA2.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "certificate signed by unknown authority")
					},
				},
				"should fail when the key is missing": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceCertFileName: certForServiceFromCA1.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "no service private key")
					},
				},
				"should fail when the certificate is missing": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:     ca1.CertPEM(),
						mtls.CAKeyFileName:      ca1.KeyPEM(),
						mtls.ServiceKeyFileName: certForServiceFromCA1.KeyPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "no service certificate")
					},
				},
				"should fail when the key is invalid": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceCertFileName: certForServiceFromCA1.CertPEM,
						mtls.ServiceKeyFileName:  []byte("invalid key"),
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "invalid private key")
					},
				},
				"should fail when the certificate is invalid": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceKeyFileName:  certForServiceFromCA1.KeyPEM,
						mtls.ServiceCertFileName: []byte("invalid cert"),
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "unparseable certificate")
					},
				},
				"should fail when the key does not match the cert": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceKeyFileName:  certForServiceFromCA2.KeyPEM,
						mtls.ServiceCertFileName: certForServiceFromCA1.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "mismatched certificate and private key")
					},
				},
				"should fail when the subject is unexpected": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceKeyFileName:  unexpectedSubjectCertFromCA1.KeyPEM,
						mtls.ServiceCertFileName: unexpectedSubjectCertFromCA1.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "unexpected certificate service type")
					},
				},
				"should fail, if for some reason, the ca cert is missing from the file map": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceKeyFileName:  certForServiceFromCA1.KeyPEM,
						mtls.ServiceCertFileName: certForServiceFromCA1.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.ErrorContains(t, err, "no CA certificate in file map")
					},
				},
				"should not fail if the ca key is missing from the file map": {
					// TODO(ROX-16206): Confirm that this behavior is correct.
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.ServiceKeyFileName:  certForServiceFromCA1.KeyPEM,
						mtls.ServiceCertFileName: certForServiceFromCA1.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.NoError(t, err)
					},
				},
				"should succeed when the certificate matches the ca": {
					ca: ca1,
					fileMap: types.SecretDataMap{
						mtls.CACertFileName:      ca1.CertPEM(),
						mtls.CAKeyFileName:       ca1.KeyPEM(),
						mtls.ServiceKeyFileName:  certForServiceFromCA1.KeyPEM,
						mtls.ServiceCertFileName: certForServiceFromCA1.CertPEM,
					},
					assert: func(t *testing.T, err error) {
						assert.NoError(t, err)
					},
				},
				// TODO(ROX-16206): Discuss tests around ca/service cert expiration, as well as "NotBefore"
			}

			for name, tt := range cases {
				t.Run(name, func(t *testing.T) {
					r := &createCentralTLSExtensionRun{
						ca: tt.ca,
					}
					switch subject {
					case mtls.ScannerSubject:
						tt.assert(t, r.validateScannerTLSData(tt.fileMap, true))
					case mtls.ScannerDBSubject:
						tt.assert(t, r.validateScannerDBTLSData(tt.fileMap, true))
					case mtls.CentralDBSubject:
						tt.assert(t, r.validateCentralDBTLSData(tt.fileMap, true))
					case mtls.ScannerV4IndexerSubject:
						tt.assert(t, r.validateScannerV4IndexerTLSData(tt.fileMap, true))
					case mtls.ScannerV4MatcherSubject:
						tt.assert(t, r.validateScannerV4MatcherTLSData(tt.fileMap, true))
					case mtls.ScannerV4DBSubject:
						tt.assert(t, r.validateScannerV4DBTLSData(tt.fileMap, true))
					}
				})
			}
		})

	}
}
