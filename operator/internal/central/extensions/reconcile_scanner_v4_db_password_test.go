package extensions

import (
	"testing"
	"time"

	"github.com/stackrox/rox/operator/internal/common/extensions"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func verifyScannerV4DBPassword(t *testing.T, data types.SecretDataMap, _ *time.Time) {
	assert.NotEmpty(t, data[extensions.ScannerV4DBPasswordKey])
}

func TestReconcileScannerV4DBPassword(t *testing.T) {
	existingScannerDBPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-v4-db-password",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("foobar"),
		},
	}

	existingMalformedScannerV4DBPassword := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-v4-db-password",
			Namespace: testutils.TestNamespace,
		},
		Data: map[string][]byte{
			"no-password": []byte("foobar"),
		},
	}

	cases := map[string]secretReconciliationTestCase{
		"When no db password secret exists and scannerV4 is disabled, no secrets should be created": {
			Spec:                   basicSpecWithScanner(false, false),
			ExpectedCreatedSecrets: nil,
		},
		"When no db password secret exists and scannerV4 is enabled, a managed secret should be created": {
			Spec: basicSpecWithScanner(false, true),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"scanner-v4-db-password": verifyScannerV4DBPassword,
			},
		},
		"When no db password secret exists and scannerV4 is enabled, and the CR is being deleted, no secrets should be created": {
			Spec:                   basicSpecWithScanner(false, true),
			Deleted:                true,
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists and scannerV4 is enabled, no secrets should be created or deleted": {
			Spec:                   basicSpecWithScanner(false, true),
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists and scannerV4 is disabled, the secret should be left intact": {
			Spec:                   basicSpecWithScanner(false, false),
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists and scannerV4 is enabled, and the CR is being deleted, the secret should be left intact": {
			Spec:                   basicSpecWithScanner(false, true),
			Deleted:                true,
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},

		// Malformed pre-existing secret
		"When a malformed unmanaged secret exists, an error is expected": {
			Spec:          basicSpecWithScanner(false, true),
			Existing:      []*v1.Secret{existingMalformedScannerV4DBPassword},
			ExpectedError: "scanner-v4-db-password secret must contain a non-empty",
		},
		"When a malformed unmanaged secret exists, no error is expected": {
			Spec:                   basicSpecWithScanner(false, false),
			Existing:               []*v1.Secret{existingMalformedScannerV4DBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When a malformed unmanaged secret exists, and the CR is being deleted, no error is expected": {
			Spec:                   basicSpecWithScanner(false, true),
			Deleted:                true,
			Existing:               []*v1.Secret{existingMalformedScannerV4DBPassword},
			ExpectedCreatedSecrets: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testSecretReconciliation(t, reconcileScannerV4DBPassword, c)
		})
	}
}
