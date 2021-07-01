package extensions

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func verifyScannerDBPassword(t *testing.T, data secretDataMap) {
	assert.NotEmpty(t, data[scannerDBPasswordKey])
}

func TestReconcileScannerDBPassword(t *testing.T) {
	existingScannerDBPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-password",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}

	existingMalformedScannerDBPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-password",
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"no-password": []byte("foobar"),
		},
	}

	cases := map[string]secretReconciliationTestCase{
		"empty-state-no-scanner": {
			ScannerEnabled:         false,
			ExpectedCreatedSecrets: nil,
		},
		"empty-state-with-scanner": {
			ScannerEnabled: true,
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"scanner-db-password": verifyScannerDBPassword,
			},
		},
		"empty-state-deleted-with-scanner": {
			ScannerEnabled:         true,
			Deleted:                true,
			ExpectedCreatedSecrets: nil,
		},
		"preexisting-with-scanner": {
			ScannerEnabled:         true,
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"preexisting-no-scanner": {
			ScannerEnabled:         false,
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"preexisting-deleted-with-scanner": {
			ScannerEnabled:         true,
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},

		// Malformed pre-existing secret
		"preexisting-malformed-with-scanner": {
			ScannerEnabled: true,
			Existing:       []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedError:  "scanner-db-password secret must contain a non-empty",
		},
		"preexisting-malformed-no-scanner": {
			ScannerEnabled:         false,
			Existing:               []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"preexisting-malformed-deleted-with-scanner": {
			ScannerEnabled:         true,
			Existing:               []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileScannerDBPassword, c)
		})
	}
}
