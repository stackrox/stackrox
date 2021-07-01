package extensions

import (
	"testing"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type secretVerifyFunc func(t *testing.T, data secretDataMap)

func verifyCentralCert(t *testing.T, data secretDataMap) {
	ca, err := certgen.LoadCAFromFileMap(data)
	require.NoError(t, err)
	assert.NoError(t, certgen.VerifyServiceCert(data, ca, mtls.CentralSubject, ""))

	_, err = certgen.LoadJWTSigningKeyFromFileMap(data)
	assert.NoError(t, err)
}

func verifyServiceCert(subj mtls.Subject) secretVerifyFunc {
	return func(t *testing.T, data secretDataMap) {
		validatingCA, err := mtls.LoadCAForValidation(data["ca.pem"])
		require.NoError(t, err)

		assert.NoError(t, certgen.VerifyServiceCert(data, validatingCA, subj, ""))
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
		"empty-state-no-scanner": {
			ScannerEnabled: false,
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls": verifyCentralCert,
			},
		},
		"empty-state-with-scanner": {
			ScannerEnabled: true,
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
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
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
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

			testSecretReconciliation(t, reconcileCentralTLS, c)
		})
	}
}
