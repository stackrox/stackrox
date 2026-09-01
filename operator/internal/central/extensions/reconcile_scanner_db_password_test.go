package extensions

import (
	"testing"
	"time"

	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func verifyScannerDBPassword(t *testing.T, data types.SecretDataMap, _ *time.Time) {
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
		"Scanner v2 is removed, no secrets should be created regardless of spec": {
			Spec:                   basicSpecWithScanner(true, false),
			ExpectedCreatedSecrets: nil,
		},
		"When no db password secret exists and scanner is disabled, no secrets should be created": {
			Spec:                   basicSpecWithScanner(false, false),
			ExpectedCreatedSecrets: nil,
		},
		"When an unmanaged db password secret exists, the secret should be left intact": {
			Spec:                   basicSpecWithScanner(true, false),
			Existing:               []*v1.Secret{existingScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
		"When a malformed unmanaged secret exists, no error is expected because scanner v2 is disabled": {
			Spec:                   basicSpecWithScanner(true, false),
			Existing:               []*v1.Secret{existingMalformedScannerDBPassword},
			ExpectedCreatedSecrets: nil,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {

			testSecretReconciliation(t, reconcileScannerDBPassword, c)
		})
	}
}
