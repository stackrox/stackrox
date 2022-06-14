package extensions

import (
	"testing"

	"github.com/stackrox/rox/operator/pkg/types"
	"github.com/stackrox/rox/operator/pkg/utils/testutils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func verifyScannerDBPassword(t *testing.T, data types.SecretDataMap) {
	assert.NotEmpty(t, data[scannerDBPasswordKey])
}

func TestReconcileScannerDBPassword(t *testing.T) {
	existingScannerDBPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-password",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}

	existingMalformedScannerDBPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-db-password",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"no-password": []byte("foobar"),
		},
	}

	cases := map[string]secretReconciliationTestCase{
		"When no db password secret exists and scanner is disabled, no secrets should be created": {
			Spec:                   basicSpecWithScanner(false),
			ExpectedCreatedSecrets: nil,
		},
		"When no db password secret exists and scanner is enabled, a managed secret should be created": {
			Spec: basicSpecWithScanner(true),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"scanner-db-password": verifyScannerDBPassword,
			},
		},
		"When no db password secret exists and scanner is enabled, and the CR is being deleted, no secrets should be created": {
			Spec:                   basicSpecWithScanner(true),
			Deleted:                true,
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists and scanner is enabled, no secrets should be created or deleted": {
			Spec:                   basicSpecWithScanner(true),
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists and scanner is disabled, the secret should be left intact": {
			Spec:                   basicSpecWithScanner(false),
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists and scanner is enabled, and the CR is being deleted, the secret should be left intact": {
			Spec:                   basicSpecWithScanner(true),
			Deleted:                true,
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},

		// Malformed pre-existing secret
		"When a malformed unmanaged secret exists, an error is expected": {
			Spec:          basicSpecWithScanner(true),
			Existing:      []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedError: "scanner-db-password secret must contain a non-empty",
		},
		"When a malformed unmanaged secret exists, no error is expected": {
			Spec:                   basicSpecWithScanner(false),
			Existing:               []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When a malformed unmanaged secret exists, and the CR is being deleted, no error is expected": {
			Spec:                   basicSpecWithScanner(true),
			Deleted:                true,
			Existing:               []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileScannerDBPassword, c)
		})
	}
}
