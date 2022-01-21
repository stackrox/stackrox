package extensions

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func verifyCentralCert(t *testing.T, data secretDataMap) {
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
	return func(t *testing.T, data secretDataMap) {
		validatingCA, err := mtls.LoadCAForValidation(data["ca.pem"])
		require.NoError(t, err)

		assert.NoError(t, certgen.VerifyServiceCert(data, validatingCA, serviceType, fileNamePrefix))
	}
}

func TestCreateCentralTLS(t *testing.T) {
	testCA, err := certgen.GenerateCA()
	require.NoError(t, err)

	centralFileMap := make(secretDataMap)
	certgen.AddCAToFileMap(centralFileMap, testCA)
	require.NoError(t, certgen.IssueCentralCert(centralFileMap, testCA))
	jwtKey, err := certgen.GenerateJWTSigningKey()
	require.NoError(t, err)
	certgen.AddJWTSigningKeyToFileMap(centralFileMap, jwtKey)

	existingCentral := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: testNamespace,
		},
		Data: centralFileMap,
	}

	scannerFileMap := make(secretDataMap)
	certgen.AddCACertToFileMap(scannerFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerFileMap, testCA, mtls.ScannerSubject, ""))

	existingScanner := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-tls",
			Namespace: testNamespace,
		},
		Data: scannerFileMap,
	}

	scannerDBFileMap := make(secretDataMap)
	certgen.AddCACertToFileMap(scannerDBFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerDBFileMap, testCA, mtls.ScannerDBSubject, ""))

	existingScannerDB := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-tls",
			Namespace: testNamespace,
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
			Other: []client.Object{&platform.SecuredCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secured-cluster-services",
					Namespace: testNamespace,
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
			// See ROX-9023.
			continue
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileCentralTLS, c)
		})
	}
}
