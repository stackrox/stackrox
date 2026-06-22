package extensions

import (
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/types"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/crs"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func verifyCRS(t *testing.T, data types.SecretDataMap, atTime *time.Time) {
	crsObj, err := crs.DeserializeSecret(string(data[crsDataKey]))
	require.NoError(t, err)
	require.Equal(t, 1, crsObj.Version)
	require.NotEmpty(t, crsObj.CAs)
	require.NotEmpty(t, crsObj.Cert)
	require.NotEmpty(t, crsObj.Key)

	block, _ := pem.Decode([]byte(crsObj.Cert))
	require.NotNil(t, block)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	subject := mtls.SubjectFromCommonName(cert.Subject.CommonName)
	assert.Equal(t, storage.ServiceType_REGISTRANT_SERVICE, subject.ServiceType)
	assert.Equal(t, centralsensor.EphemeralInitCertClusterID, subject.Identifier)

	if atTime != nil {
		run := &createCentralTLSExtensionRun{currentTime: *atTime}
		assert.NoError(t, run.checkCertRenewal(cert))
	}
}

func expectedSecretsWithCRS(secrets map[string]secretVerifyFunc) map[string]secretVerifyFunc {
	if secrets == nil {
		secrets = make(map[string]secretVerifyFunc)
	} else {
		copied := make(map[string]secretVerifyFunc, len(secrets)+1)
		for name, verify := range secrets {
			copied[name] = verify
		}
		secrets = copied
	}
	secrets[clusterRegistrationSecretName] = verifyCRS
	return secrets
}

func buildCRSData(t *testing.T, ca mtls.CA, opts ...mtls.IssueCertOption) types.SecretDataMap {
	t.Helper()
	crsID := uuid.NewV4()
	subject := mtls.NewInitSubject(centralsensor.EphemeralInitCertClusterID, storage.ServiceType_REGISTRANT_SERVICE, crsID)
	issuedCert, err := ca.IssueCertForSubject(subject, append([]mtls.IssueCertOption{mtls.WithValidityExpiringInHours()}, opts...)...)
	require.NoError(t, err)
	crsObj := &crs.CRS{
		Version: 1,
		CAs:     []string{string(ca.CertPEM())},
		Cert:    string(issuedCert.CertPEM),
		Key:     string(issuedCert.KeyPEM),
	}
	serialized, err := crs.SerializeSecret(crsObj)
	require.NoError(t, err)
	return types.SecretDataMap{crsDataKey: []byte(serialized)}
}

func TestReconcileCRSSecret(t *testing.T) {
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

	siblingSecuredCluster := &platform.SecuredCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secured-cluster-services",
			Namespace: testutils.TestNamespace,
		},
	}

	cases := map[string]secretReconciliationTestCase{
		"When no secured cluster exists, operator-managed CRS secret should still be created": {
			Spec:            basicSpecWithScanner(false, false),
			ExistingManaged: []*v1.Secret{existingCentral, existingCentralDB},
			ExpectedCreatedSecrets: expectedSecretsWithCRS(map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
			}),
		},
		"When CRS certificate is past half validity, it should be renewed": {
			Spec: basicSpecWithScanner(false, false),
			ExistingManaged: []*v1.Secret{
				existingCentral,
				existingCentralDB,
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterRegistrationSecretName,
						Namespace: testutils.TestNamespace,
					},
					Data: buildCRSData(t, testCA,
						mtls.WithValidityNotBefore(time.Now().Add(-4*time.Hour)),
						mtls.WithValidityNotAfter(time.Now().Add(-1*time.Hour)),
					),
				},
			},
			ExpectedCreatedSecrets: expectedSecretsWithCRS(map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
			}),
		},
		"When creating cluster-registration-secret fails, an error should be returned": {
			Spec:                   basicSpecWithScanner(false, false),
			ExistingManaged:        []*v1.Secret{existingCentral, existingCentralDB},
			InterceptedK8sAPICalls: creatingSecretFails(clusterRegistrationSecretName),
			ExpectedError:          "reconciling cluster-registration-secret",
		},
		"When a sibling secured cluster exists, CRS is still created the same way": {
			Spec: basicSpecWithScanner(false, false),
			Other: []ctrlClient.Object{
				siblingSecuredCluster,
			},
			ExistingManaged: []*v1.Secret{existingCentral, existingCentralDB},
			ExpectedCreatedSecrets: expectedSecretsWithCRS(map[string]secretVerifyFunc{
				"central-tls":    verifyCentralCert,
				"central-db-tls": verifyCentralServiceCert(storage.ServiceType_CENTRAL_DB_SERVICE),
			}),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			testSecretReconciliation(t, reconcileCentralTLS, c)
		})
	}
}

func Test_validateCRSData(t *testing.T) {
	testCA, err := certgen.GenerateCA()
	require.NoError(t, err)

	validCRS := buildCRSData(t, testCA)
	run := &createCentralTLSExtensionRun{
		ca:          testCA,
		currentTime: time.Now(),
	}
	assert.NoError(t, run.validateCRSData(validCRS, true))

	otherCA, err := certgen.GenerateCA()
	require.NoError(t, err)
	wrongCA := buildCRSData(t, otherCA)
	assert.ErrorContains(t, run.validateCRSData(wrongCA, true), "CRS CA does not match current CA")
}
