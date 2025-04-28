package extensions

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	pkgErrors "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	commonLabels "github.com/stackrox/rox/operator/internal/common/labels"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
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
	assert.NoError(t, certgen.VerifyServiceCertAndKey(data, "", ca, storage.ServiceType_CENTRAL_SERVICE))

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

		assert.NoError(t, certgen.VerifyServiceCertAndKey(data, fileNamePrefix, validatingCA, serviceType))
	}
}

// Inspired by certgen.GenerateCA.
func generateInvalidCA() (mtls.CA, error) {
	serial, err := mtls.RandomSerial()
	if err != nil {
		return nil, pkgErrors.Wrap(err, "could not generate a serial number")
	}
	req := csr.CertificateRequest{
		CN:           mtls.ServiceCACommonName + " this makes it invalid",
		KeyRequest:   csr.NewKeyRequest(),
		SerialNumber: serial.String(),
	}
	caCert, _, caKey, err := initca.New(&req)
	if err != nil {
		return nil, pkgErrors.Wrap(err, "could not generate keypair")
	}
	return mtls.LoadCAForSigning(caCert, caKey)
}

func generateTestCertWithValidity(t *testing.T, notBeforeStr, notAfterStr string) *x509.Certificate {
	t.Helper()
	notBefore, err := time.Parse(time.RFC3339, notBeforeStr)
	require.NoError(t, err)
	notAfter, err := time.Parse(time.RFC3339, notAfterStr)
	require.NoError(t, err)
	return &x509.Certificate{
		NotBefore: notBefore,
		NotAfter:  notAfter,
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

	centralFileMapWithInvalidLeaf := make(types.SecretDataMap)
	certgen.AddCAToFileMap(centralFileMapWithInvalidLeaf, testCA)
	unrelatedTestCA, err := certgen.GenerateCA()
	require.NoError(t, err)
	// Resulting cert will not match CA and thus be up for replacement.
	require.NoError(t, certgen.IssueCentralCert(centralFileMapWithInvalidLeaf, unrelatedTestCA))
	certgen.AddJWTSigningKeyToFileMap(centralFileMapWithInvalidLeaf, jwtKey)

	existingCentralWithInvalidLeaf := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: testutils.TestNamespace,
			Labels:    commonLabels.DefaultLabels(),
		},
		Data: centralFileMapWithInvalidLeaf,
	}

	centralFileMapWithMissingCAKey := make(types.SecretDataMap)
	certgen.AddCAToFileMap(centralFileMapWithMissingCAKey, testCA)
	delete(centralFileMapWithMissingCAKey, mtls.CAKeyFileName)
	require.NoError(t, certgen.IssueCentralCert(centralFileMapWithMissingCAKey, testCA))
	certgen.AddJWTSigningKeyToFileMap(centralFileMapWithMissingCAKey, jwtKey)

	existingCentralWithMissingCAKey := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: testutils.TestNamespace,
			Labels:    commonLabels.DefaultLabels(),
		},
		Data: centralFileMapWithMissingCAKey,
	}

	centralFileMapWithInvalidCA := make(types.SecretDataMap)
	invalidCA, err := generateInvalidCA()
	require.NoError(t, err)
	certgen.AddCAToFileMap(centralFileMapWithInvalidCA, invalidCA)
	require.NoError(t, certgen.IssueCentralCert(centralFileMapWithInvalidCA, invalidCA))
	certgen.AddJWTSigningKeyToFileMap(centralFileMapWithInvalidCA, jwtKey)

	existingCentralWithInvalidCA := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: centralFileMapWithInvalidCA,
	}

	centralFileMapWithCorruptCA := make(types.SecretDataMap)
	centralFileMapWithCorruptCA[mtls.CACertFileName] = []byte("corrupt cert")
	centralFileMapWithCorruptCA[mtls.CAKeyFileName] = []byte("corrupt key")
	require.NoError(t, certgen.IssueCentralCert(centralFileMapWithCorruptCA, testCA))
	certgen.AddJWTSigningKeyToFileMap(centralFileMap, jwtKey)

	existingCentralWithCorruptCA := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: centralFileMapWithCorruptCA,
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

	scannerV4IndexerFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(scannerV4IndexerFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerV4IndexerFileMap, testCA, mtls.ScannerV4IndexerSubject, ""))

	existingScannerV4Indexer := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-v4-indexer-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: scannerV4IndexerFileMap,
	}

	scannerV4MatcherFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(scannerV4MatcherFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerV4MatcherFileMap, testCA, mtls.ScannerV4MatcherSubject, ""))

	existingScannerV4Matcher := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-v4-matcher-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: scannerV4MatcherFileMap,
	}

	scannerV4DBFileMap := make(types.SecretDataMap)
	certgen.AddCACertToFileMap(scannerV4DBFileMap, testCA)
	require.NoError(t, certgen.IssueServiceCert(scannerV4DBFileMap, testCA, mtls.ScannerV4DBSubject, ""))

	existingScannerV4DB := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scanner-v4-db-tls",
			Namespace: testutils.TestNamespace,
		},
		Data: scannerV4DBFileMap,
	}

	cases := map[string]secretReconciliationTestCase{
		"When no secrets exist and scanner is disabled, a managed central-tls and central-db-tls secrets should be created": {
			Spec: basicSpecWithScanner(false, false),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
			},
		},
		"When no secrets exist and scanner is disabled but secured cluster exists, a managed central-tls secret and init bundle secrets should be created": {
			Spec: basicSpecWithScanner(false, false),
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
			Spec: basicSpecWithScanner(true, true),
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":            verifyCentralCert,
				"central-db-tls":         verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
				"scanner-tls":            verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
				"scanner-db-tls":         verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
				"scanner-v4-indexer-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
				"scanner-v4-matcher-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_MATCHER_SERVICE),
				"scanner-v4-db-tls":      verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_DB_SERVICE),
			},
		},
		"When a managed central-tls secret with valid CA but invalid service cert exists, it should be fixed": {
			Spec:            basicSpecWithScanner(false, false),
			ExistingManaged: []*v1.Secret{existingCentralWithInvalidLeaf, existingCentralDB},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
			},
		},
		"When a managed central-tls secret with missing CA key exists, it should fail": {
			Spec:            basicSpecWithScanner(false, false),
			ExistingManaged: []*v1.Secret{existingCentralWithMissingCAKey, existingCentralDB},
			ExpectedError:   "malformed secret (ca.pem present but ca-key.pem missing), please delete it",
		},
		"When a managed central-tls secret with invalid CA exists, it should fail": {
			Spec:            basicSpecWithScanner(false, false),
			ExistingManaged: []*v1.Secret{existingCentralWithInvalidCA, existingCentralDB},
			ExpectedError:   "invalid properties of CA in the existing secret, please delete it to allow re-generation: invalid certificate common name",
		},
		"When a managed central-tls secret with corrupt CA exists, it should fail": {
			Spec:            basicSpecWithScanner(false, false),
			ExistingManaged: []*v1.Secret{existingCentralWithCorruptCA, existingCentralDB},
			ExpectedError:   "invalid CA in the existing secret, please delete it to allow re-generation: tls: failed to find any PEM data in certificate input",
		},
		"When a valid unmanaged central-tls and central-db-tls secrets exist and scanner is disabled, no further secrets should be created": {
			Spec:     basicSpecWithScanner(false, false),
			Existing: []*v1.Secret{existingCentral, existingCentralDB},
		},
		"When a valid unmanaged central-tls and central-db-tls secrets exist and scanner is enabled, managed secrets should be created for scanner": {
			Spec:     basicSpecWithScanner(true, true),
			Existing: []*v1.Secret{existingCentral, existingCentralDB},
			ExpectedCreatedSecrets: map[string]secretVerifyFunc{
				"scanner-tls":            verifyCentralServiceCert(storage.ServiceType_SCANNER_SERVICE),
				"scanner-db-tls":         verifyCentralServiceCert(storage.ServiceType_SCANNER_DB_SERVICE),
				"scanner-v4-indexer-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
				"scanner-v4-matcher-tls": verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_MATCHER_SERVICE),
				"scanner-v4-db-tls":      verifyCentralServiceCert(storage.ServiceType_SCANNER_V4_DB_SERVICE),
			},
		},
		"When valid unmanaged secrets exist for everything and scanner is disabled, no secrets should be created or deleted": {
			Spec: basicSpecWithScanner(false, false),
			Existing: []*v1.Secret{
				existingCentral, existingCentralDB, existingScanner, existingScannerDB,
				existingScannerV4Indexer, existingScannerV4Matcher, existingScannerV4DB,
			},
		},
		"When valid unmanaged secrets exist for everything and scanner is enabled, no secrets should be created or deleted": {
			Spec: basicSpecWithScanner(true, true),
			Existing: []*v1.Secret{
				existingCentral, existingCentralDB, existingScanner, existingScannerDB,
				existingScannerV4Indexer, existingScannerV4Matcher, existingScannerV4DB,
			},
		},
		"When creating a new central-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, false),
			InterceptedK8sAPICalls: creatingSecretFails("central-tls"),
			ExpectedError:          "reconciling central-tls secret",
		},
		"When creating a new central-db-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, false),
			InterceptedK8sAPICalls: creatingSecretFails("central-db-tls"),
			ExpectedError:          "reconciling central-db-tls secret",
		},
		"When creating a new scanner-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(true, false),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-tls"),
			ExpectedError:          "reconciling scanner-tls secret",
		},
		"When creating a new scanner-db-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(true, false),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-db-tls"),
			ExpectedError:          "reconciling scanner-db-tls secret",
		},
		"When creating a new scanner-v4-indexer-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, true),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-v4-indexer-tls"),
			ExpectedError:          "reconciling scanner-v4-indexer-tls secret",
		},
		"When creating a new scanner-v4-matcher-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, true),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-v4-matcher-tls"),
			ExpectedError:          "reconciling scanner-v4-matcher-tls secret",
		},
		"When creating a new scanner-v4-db-tls secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, true),
			InterceptedK8sAPICalls: creatingSecretFails("scanner-v4-db-tls"),
			ExpectedError:          "reconciling scanner-v4-db-tls secret",
		},
		"When getting an existing central-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, false),
			InterceptedK8sAPICalls: gettingSecretFails("central-tls"),
			ExpectedError:          "reconciling central-tls secret",
		},
		"When getting an existing central-db-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, false),
			InterceptedK8sAPICalls: gettingSecretFails("central-db-tls"),
			ExpectedError:          "reconciling central-db-tls secret",
		},
		"When getting an existing scanner-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(true, false),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-tls"),
			ExpectedError:          "reconciling scanner-tls secret",
		},
		"When getting an existing scanner-db-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(true, false),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-db-tls"),
			ExpectedError:          "reconciling scanner-db-tls secret",
		},
		"When getting an existing scanner-v4-indexer-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, true),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-v4-indexer-tls"),
			ExpectedError:          "reconciling scanner-v4-indexer-tls secret",
		},
		"When getting an existing scanner-v4-matcher-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, true),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-v4-matcher-tls"),
			ExpectedError:          "reconciling scanner-v4-matcher-tls secret",
		},
		"When getting an existing scanner-v4-db-tls secret fails with a non-404 error, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, true),
			InterceptedK8sAPICalls: gettingSecretFails("scanner-v4-db-tls"),
			ExpectedError:          "reconciling scanner-v4-db-tls secret",
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
		"When deleting an existing scanner-v4-indexer-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScannerV4Indexer},
			InterceptedK8sAPICalls: deletingSecretFails("scanner-v4-indexer-tls"),
			ExpectedError:          "reconciling scanner-v4-indexer-tls secret",
		},
		"When deleting an existing scanner-v4-matcher-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScannerV4Matcher},
			InterceptedK8sAPICalls: deletingSecretFails("scanner-v4-matcher-tls"),
			ExpectedError:          "reconciling scanner-v4-matcher-tls secret",
		},
		"When deleting an existing scanner-v4-db-tls secret fails, an error should be returned": {
			Deleted:                true,
			ExistingManaged:        []*v1.Secret{existingScannerV4DB},
			InterceptedK8sAPICalls: deletingSecretFails("scanner-v4-db-tls"),
			ExpectedError:          "reconciling scanner-v4-db-tls secret",
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

func Test_checkCertRenewal(t *testing.T) {
	cases := map[string]struct {
		now       string
		notBefore string
		notAfter  string
		wantErr   assert.ErrorAssertionFunc
	}{
		"should reject certificate that becomes invalid before it becomes valid": {
			now:       "2021-02-11T11:59:00.000Z",
			notBefore: "2021-02-11T23:59:59.000Z",
			notAfter:  "2021-02-11T00:00:00.000Z",
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate expires at 2021-02-11 00:00:00 +0000 UTC before it begins to be valid at 2021-02-11 23:59:59 +0000 UTC")
			},
		},
		"should NOT return error when the certificate remains valid": {
			now:       "2021-02-11T11:59:00.000Z",
			notBefore: "2021-02-11T00:00:00.000Z",
			notAfter:  "2021-02-11T23:59:59.000Z",
			wantErr:   assert.NoError,
		},
		"should NOT return error when the certificate is valid for an extremely short time": {
			now:       "2021-02-11T00:00:00.001Z",
			notBefore: "2021-02-11T00:00:00.000Z",
			notAfter:  "2021-02-11T00:00:00.004Z",
			wantErr:   assert.NoError,
		},
		"should return error when the certificate is already expired": {
			now:       "2021-02-11T12:00:00.000Z",
			notBefore: "2021-02-11T00:00:00.000Z",
			notAfter:  "2021-02-11T11:00:00.000Z",
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate expired at 2021-02-11 11:00:00 +0000 UTC")
			},
		},
		"should return error when the certificate lifetime is not started": {
			now:       "2021-02-11T12:00:00.000Z",
			notBefore: "2021-02-11T22:00:00.000Z",
			notAfter:  "2021-02-11T23:59:59.000Z",
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate lifetime start 2021-02-11 22:00:00 +0000 UTC is in the future")
			},
		},
		"should return error when the certificate expires soon": {
			now:       "2021-02-11T12:00:00.000Z",
			notBefore: "2021-02-11T00:00:00.000Z",
			notAfter:  "2021-02-11T12:30:00.000Z",
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate is past half of its validity, 2021-02-11 06:15:00 +0000 UTC")
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cert := generateTestCertWithValidity(t, c.notBefore, c.notAfter)
			now, err := time.Parse(time.RFC3339, c.now)
			require.NoError(t, err)
			r := &createCentralTLSExtensionRun{currentTime: now}

			c.wantErr(t, r.checkCertRenewal(cert), fmt.Sprintf("checkCertRenewal(%v, %v)", cert, now))
		})
	}
}

func Test_checkCARotation(t *testing.T) {
	cases := map[string]struct {
		now                string
		primaryNotBefore   string
		primaryNotAfter    string
		secondaryNotBefore string
		secondaryNotAfter  string
		wantAction         CARotationAction
		wantErr            assert.ErrorAssertionFunc
	}{
		"should return error if primary is nil": {
			now:        "2026-01-01T00:00:00Z",
			wantAction: CARotateNoAction,
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "primary CA must not be nil")
			},
		},
		"should return error if primary has invalid validity range": {
			now:              "2026-01-01T00:00:00Z",
			primaryNotBefore: "2030-01-01T00:00:00Z",
			primaryNotAfter:  "2025-01-01T00:00:00Z",
			wantAction:       CARotateNoAction,
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate expires at")
			},
		},
		"should return error if primary is not yet valid": {
			now:              "2024-01-01T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       CARotateNoAction,
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate lifetime start")
			},
		},
		"should return error if primary is expired": {
			now:              "2031-01-01T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       CARotateNoAction,
			wantErr: func(t assert.TestingT, err error, _ ...interface{}) bool {
				return assert.ErrorContains(t, err, "certificate expired")
			},
		},
		"should return no action in first 3/5 of validity": {
			now:              "2026-06-01T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       CARotateNoAction,
			wantErr:          assert.NoError,
		},
		"should add secondary after 3/5 of validity": {
			now:              "2028-01-02T00:00:00Z",
			primaryNotBefore: "2025-01-01T00:00:00Z",
			primaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:       CARotateAddSecondary,
			wantErr:          assert.NoError,
		},
		"should promote secondary after 4/5 of validity": {
			now:                "2029-01-02T00:00:00Z",
			primaryNotBefore:   "2025-01-01T00:00:00Z",
			primaryNotAfter:    "2030-01-01T00:00:00Z",
			secondaryNotBefore: "2028-01-01T00:00:00Z",
			secondaryNotAfter:  "2033-01-01T00:00:00Z",
			wantAction:         CARotatePromoteSecondary,
			wantErr:            assert.NoError,
		},
		"should delete expired secondary": {
			now:                "2031-01-02T00:00:00Z",
			primaryNotBefore:   "2028-01-01T00:00:00Z",
			primaryNotAfter:    "2033-01-01T00:00:00Z",
			secondaryNotBefore: "2025-01-01T00:00:00Z",
			secondaryNotAfter:  "2030-01-01T00:00:00Z",
			wantAction:         CARotateDeleteSecondary,
			wantErr:            assert.NoError,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, c.now)
			require.NoError(t, err)

			var primary *x509.Certificate
			if c.primaryNotBefore != "" && c.primaryNotAfter != "" {
				primary = generateTestCertWithValidity(t, c.primaryNotBefore, c.primaryNotAfter)
			}

			var secondary *x509.Certificate
			if c.secondaryNotBefore != "" && c.secondaryNotAfter != "" {
				secondary = generateTestCertWithValidity(t, c.secondaryNotBefore, c.secondaryNotAfter)
			}

			r := &createCentralTLSExtensionRun{currentTime: now}
			action, err := r.checkCARotation(primary, secondary)

			assert.Equal(t, c.wantAction, action)
			c.wantErr(t, err)
		})
	}
}

func TestGenerateCentralTLSData_Rotation(t *testing.T) {
	type testCase struct {
		name            string
		action          CARotationAction
		additionalSetup func(t *testing.T, old types.SecretDataMap)
		assert          func(t *testing.T, old, new types.SecretDataMap)
	}

	cases := []testCase{
		{
			name:   "add secondary CA",
			action: CARotateAddSecondary,
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Contains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be present")
				require.Contains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be present")
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA should be unchanged")
			},
		},
		{
			name:   "promote secondary CA",
			action: CARotatePromoteSecondary,
			additionalSetup: func(t *testing.T, old types.SecretDataMap) {
				secondary, err := certgen.GenerateCA()
				require.NoError(t, err)
				certgen.AddSecondaryCAToFileMap(old, secondary)
			},
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Contains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be present")
				require.Contains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be present")
				require.NotEqual(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA should have changed")
				require.Equal(t, new[mtls.SecondaryCACertFileName], old[mtls.CACertFileName],
					"secondary CA cert should be the old primary CA cert")
				require.Equal(t, new[mtls.SecondaryCAKeyFileName], old[mtls.CAKeyFileName],
					"secondary CA key should be the old primary CA key")
				require.Equal(t, new[mtls.CACertFileName], old[mtls.SecondaryCACertFileName],
					"primary CA cert should be the old secondary CA cert")
				require.Equal(t, new[mtls.CAKeyFileName], old[mtls.SecondaryCAKeyFileName],
					"primary CA key should be the old secondary CA key")
				require.Contains(t, new, mtls.ServiceCertFileName, "central cert should be present")
				require.Contains(t, new, mtls.ServiceKeyFileName, "central cert should be present")
			},
		},
		{
			name:   "delete secondary CA",
			action: CARotateDeleteSecondary,
			additionalSetup: func(t *testing.T, old types.SecretDataMap) {
				secondary, err := certgen.GenerateCA()
				require.NoError(t, err)
				certgen.AddSecondaryCAToFileMap(old, secondary)
			},
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA cert should be unchanged")
				require.Equal(t, old[mtls.CAKeyFileName], new[mtls.CAKeyFileName], "primary CA key should be unchanged")
				require.NotContains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be removed")
				require.NotContains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be removed")
			},
		},
		{
			name:   "no rotation action, secondary CA not present",
			action: CARotateNoAction,
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA cert should be unchanged")
				require.Equal(t, old[mtls.CAKeyFileName], new[mtls.CAKeyFileName], "primary CA key should be unchanged")
				require.NotContains(t, new, mtls.SecondaryCACertFileName, "secondary CA cert should be present")
				require.NotContains(t, new, mtls.SecondaryCAKeyFileName, "secondary CA key should be present")
				require.NotEqual(t, old[mtls.ServiceCertFileName], new[mtls.ServiceCertFileName], "central cert should be reissued")
				require.NotEqual(t, old[mtls.ServiceKeyFileName], new[mtls.ServiceKeyFileName], "central key should be reissued")
			},
		},
		{
			name:   "no rotation action, secondary CA present",
			action: CARotateNoAction,
			additionalSetup: func(t *testing.T, old types.SecretDataMap) {
				secondary, err := certgen.GenerateCA()
				require.NoError(t, err)
				certgen.AddSecondaryCAToFileMap(old, secondary)
			},
			assert: func(t *testing.T, old, new types.SecretDataMap) {
				require.Equal(t, old[mtls.CACertFileName], new[mtls.CACertFileName], "primary CA cert should be unchanged")
				require.Equal(t, old[mtls.CAKeyFileName], new[mtls.CAKeyFileName], "primary CA key should be unchanged")
				require.Equal(t, old[mtls.SecondaryCACertFileName], new[mtls.SecondaryCACertFileName], "secondary CA cert should be unchanged")
				require.Equal(t, old[mtls.SecondaryCAKeyFileName], new[mtls.SecondaryCAKeyFileName], "secondary CA key should be unchanged")
				require.NotEqual(t, old[mtls.ServiceCertFileName], new[mtls.ServiceCertFileName], "central cert should be reissued")
				require.NotEqual(t, old[mtls.ServiceKeyFileName], new[mtls.ServiceKeyFileName], "central key should be reissued")
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			primary, err := certgen.GenerateCA()
			require.NoError(t, err)
			oldFileMap := make(types.SecretDataMap)
			certgen.AddCAToFileMap(oldFileMap, primary)
			err = certgen.IssueCentralCert(oldFileMap, primary, mtls.WithNamespace("stackrox"))
			require.NoError(t, err)

			if tt.additionalSetup != nil {
				tt.additionalSetup(t, oldFileMap)
			}

			r := &createCentralTLSExtensionRun{
				centralObj:       &platform.Central{ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"}},
				currentTime:      time.Now(),
				caRotationAction: tt.action,
				ca:               primary,
			}

			newFileMap, err := r.generateCentralTLSData(oldFileMap)
			require.NoError(t, err)
			require.NotNil(t, newFileMap)

			tt.assert(t, oldFileMap, newFileMap)
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
				assert.ErrorContains(t, err, "no service certificate for CENTRAL_SERVICE in file map")
			},
		},
		"should fail when the service key is missing": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:      ca1.CertPEM(),
				mtls.CAKeyFileName:       ca1.KeyPEM(),
				mtls.ServiceCertFileName: centralCertFromCA1.CertPEM,
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "no service private key for CENTRAL_SERVICE in file map")
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
		"should fail when the secondary CA key is missing": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:          ca1.CertPEM(),
				mtls.CAKeyFileName:           ca1.KeyPEM(),
				mtls.ServiceCertFileName:     centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:      centralCertFromCA1.KeyPEM,
				mtls.SecondaryCACertFileName: ca2.CertPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "no CA key in file map")
				assert.ErrorContains(t, err, "loading secondary CA failed")
			},
		},
		"should fail when the secondary CA key is invalid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:          ca1.CertPEM(),
				mtls.CAKeyFileName:           ca1.KeyPEM(),
				mtls.ServiceCertFileName:     centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:      centralCertFromCA1.KeyPEM,
				mtls.SecondaryCACertFileName: ca2.CertPEM(),
				mtls.SecondaryCAKeyFileName:  []byte("invalid"),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to find any PEM data in key input")
				assert.ErrorContains(t, err, "loading secondary CA failed")
			},
		},
		"should fail when the secondary CA cert is invalid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:          ca1.CertPEM(),
				mtls.CAKeyFileName:           ca1.KeyPEM(),
				mtls.ServiceCertFileName:     centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:      centralCertFromCA1.KeyPEM,
				mtls.SecondaryCACertFileName: []byte("invalid"),
				mtls.SecondaryCAKeyFileName:  ca2.KeyPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "failed to find any PEM data in certificate input")
				assert.ErrorContains(t, err, "loading secondary CA failed")
			},
		},
		"should fail when the secondary CA has an invalid common name": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:          ca1.CertPEM(),
				mtls.CAKeyFileName:           ca1.KeyPEM(),
				mtls.ServiceCertFileName:     centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:      centralCertFromCA1.KeyPEM,
				mtls.SecondaryCACertFileName: caNotFromACS.CertPEM(),
				mtls.SecondaryCAKeyFileName:  caNotFromACS.KeyPEM(),
			},
			assert: func(t *testing.T, err error) {
				assert.ErrorContains(t, err, "invalid certificate common name")
			},
		},
		"should succeed when the primary and secondary CAs are valid": {
			fileMap: types.SecretDataMap{
				mtls.CACertFileName:          ca1.CertPEM(),
				mtls.CAKeyFileName:           ca1.KeyPEM(),
				mtls.ServiceCertFileName:     centralCertFromCA1.CertPEM,
				mtls.ServiceKeyFileName:      centralCertFromCA1.KeyPEM,
				mtls.SecondaryCACertFileName: ca2.CertPEM(),
				mtls.SecondaryCAKeyFileName:  ca2.KeyPEM(),
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
			r := &createCentralTLSExtensionRun{currentTime: time.Now()}
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
						ca:          tt.ca,
						currentTime: time.Now(),
					}
					switch subject {
					case mtls.ScannerSubject:
						tt.assert(t, r.validateScannerTLSData(tt.fileMap, true))
					case mtls.ScannerDBSubject:
						tt.assert(t, r.validateScannerDBTLSData(tt.fileMap, true))
					case mtls.CentralDBSubject:
						tt.assert(t, r.validateCentralDBTLSData(tt.fileMap, true))
					}
				})
			}
		})

	}
}
