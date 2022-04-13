package extensions

import (
	"crypto/x509"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/stackrox/generated/storage"
	platform "github.com/stackrox/stackrox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/stackrox/operator/pkg/types"
	"github.com/stackrox/stackrox/operator/pkg/utils/testutils"
	"github.com/stackrox/stackrox/pkg/certgen"
	"github.com/stackrox/stackrox/pkg/mtls"
	"github.com/stackrox/stackrox/pkg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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

	scannerFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(scannerFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerSubject, ""))

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
		"When no secrets exist and scanner is disabled, a managed central-tls secret should be created": {
			Spec: basicSpecWithScanner(false),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls": verifyCentralCert,
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
				"admission-control-tls": verifySecuredClusterServiceCert(storage.ServiceType_ADMISSION_CONTROL_SERVICE),
				"collector-tls":         verifySecuredClusterServiceCert(storage.ServiceType_COLLECTOR_SERVICE),
				"sensor-tls":            verifySecuredClusterServiceCert(storage.ServiceType_SENSOR_SERVICE),
			},
		},
		"When no secrets exist and scanner is enabled, all managed secrets should be created": {
			Spec: basicSpecWithScanner(true),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"scanner-tls":    verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
				"scanner-db-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
			},
		},
		"When a valid unmanaged central-tls secret exists and scanner is disabled, no further secrets should be created": {
			Spec:     basicSpecWithScanner(false),
			Existing: []*v1.Secret{existingCentral},
		},
		"When a valid unmanaged central-tls secret exists and scanner is enabled, managed secrets should be created for scanner": {
			Spec:     basicSpecWithScanner(true),
			Existing: []*v1.Secret{existingCentral},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"scanner-tls":    verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
				"scanner-db-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
			},
		},
		"When valid unmanaged secrets exist for everything and scanner is disabled, no secrets should be created or deleted": {
			Spec:     basicSpecWithScanner(false),
			Existing: []*v1.Secret{existingCentral, existingScanner, existingScannerDB},
		},
		"When valid unmanaged secrets exist for everything and scanner is enabled, no secrets should be created or deleted": {
			Spec:     basicSpecWithScanner(true),
			Existing: []*v1.Secret{existingCentral, existingScanner, existingScannerDB},
		},
		// TODO(ROX-7416): Test error cases
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
