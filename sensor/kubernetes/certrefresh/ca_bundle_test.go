package certrefresh

import (
	"context"
	"crypto/x509"
	"testing"

	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/x509utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestConvertCertsToPEM(t *testing.T) {
	testCases := []struct {
		name          string
		certs         []*x509.Certificate
		expectedCount int
		shouldFail    bool
		expectedNames []string
	}{
		{
			name:       "empty input",
			certs:      []*x509.Certificate{},
			shouldFail: true,
		},
		{
			name:          "single certificate",
			certs:         []*x509.Certificate{createTestCertificate(t, "Single CA")},
			expectedCount: 1,
			expectedNames: []string{"Single CA"},
		},
		{
			name:          "two certificates",
			certs:         []*x509.Certificate{createTestCertificate(t, "Primary CA"), createTestCertificate(t, "Secondary CA")},
			expectedCount: 2,
			expectedNames: []string{"Primary CA", "Secondary CA"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pemData, err := convertCertsToPEM(tc.certs)

			if tc.shouldFail {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, pemData)

			// Verify that certificates are in PEM format
			pemString := string(pemData)
			assert.Contains(t, pemString, "-----BEGIN CERTIFICATE-----")
			assert.Contains(t, pemString, "-----END CERTIFICATE-----")

			// Convert back to x509 and verify content
			parsedCerts, err := x509utils.ConvertPEMTox509Certs(pemData)
			require.NoError(t, err)
			require.Len(t, parsedCerts, tc.expectedCount)

			// Verify certificate names
			for _, expectedName := range tc.expectedNames {
				found := false
				for _, cert := range parsedCerts {
					if cert.Subject.CommonName == expectedName {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected certificate with name %s not found", expectedName)
			}
		})
	}
}

func TestCreateTLSCABundleConfigMap(t *testing.T) {
	testCases := []struct {
		name          string
		certCount     int
		expectedNames []string
		shouldFail    bool
	}{
		{
			name:       "empty certificates",
			certCount:  0,
			shouldFail: true,
		},
		{
			name:          "single certificate",
			certCount:     1,
			expectedNames: []string{"Test CA"},
		},
		{
			name:          "multiple certificates",
			certCount:     2,
			expectedNames: []string{"Primary CA", "Secondary CA"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POD_NAMESPACE", "test-namespace")
			t.Setenv("POD_NAME", "test-sensor-pod")
			k8sClient := fake.NewSimpleClientset(
				createTestPod("test-sensor-pod", "test-namespace", "test-rs"),
				createTestReplicaSet("test-rs", "test-namespace", "test-deployment"),
			)

			var certs []*x509.Certificate
			switch tc.certCount {
			case 1:
				certs = []*x509.Certificate{createTestCertificate(t, "Test CA")}
			case 2:
				certs = []*x509.Certificate{
					createTestCertificate(t, "Primary CA"),
					createTestCertificate(t, "Secondary CA"),
				}
			}

			// Test both FromCerts and FromPEM functions
			for _, testedFunc := range []string{"from_certs", "from_pem"} {
				t.Run(testedFunc, func(t *testing.T) {
					ctx := context.Background()

					var err error
					switch testedFunc {
					case "from_certs":
						err = CreateTLSCABundleConfigMapFromCerts(ctx, certs, k8sClient)
					case "from_pem":
						var pemData []byte
						if tc.certCount == 0 {
							pemData = []byte{}
						} else {
							pemData, err = convertCertsToPEM(certs)
							require.NoError(t, err)
						}
						err = CreateTLSCABundleConfigMapFromPEM(ctx, pemData, k8sClient)
					}

					if tc.shouldFail {
						assert.Error(t, err)
						return
					}

					require.NoError(t, err)

					configMap, err := k8sClient.CoreV1().ConfigMaps("test-namespace").Get(ctx,
						pkgKubernetes.TLSCABundleConfigMapName, metav1.GetOptions{})
					require.NoError(t, err)
					require.NotNil(t, configMap)

					verifyConfigMapData(t, configMap.Data, tc.expectedNames)
					assert.Equal(t, pkgKubernetes.TLSCABundleConfigMapName, configMap.Name)
					assert.Equal(t, "test-namespace", configMap.Namespace)
					assert.Contains(t, configMap.Annotations, tlsCABundleAnnotationKey)
					assert.Equal(t, "sensor", configMap.Labels["app.stackrox.io/managed-by"])
					assert.Len(t, configMap.OwnerReferences, 1)
					assert.Equal(t, "Deployment", configMap.OwnerReferences[0].Kind)

					// Clean up for the next test
					err = k8sClient.CoreV1().ConfigMaps("test-namespace").Delete(ctx,
						pkgKubernetes.TLSCABundleConfigMapName, metav1.DeleteOptions{})
					require.NoError(t, err)
				})
			}
		})
	}
}

func TestCreateTLSCABundleConfigMapUpdate(t *testing.T) {
	t.Setenv("POD_NAMESPACE", "test-namespace")
	t.Setenv("POD_NAME", "test-sensor-pod")

	k8sClient := fake.NewSimpleClientset(
		createTestPod("test-sensor-pod", "test-namespace", "test-rs"),
		createTestReplicaSet("test-rs", "test-namespace", "test-deployment"),
	)

	ctx := context.Background()
	cert1 := createTestCertificate(t, "First CA")
	cert2 := createTestCertificate(t, "Second CA")

	err := CreateTLSCABundleConfigMapFromCerts(ctx, []*x509.Certificate{cert1}, k8sClient)
	require.NoError(t, err)

	configMap, err := k8sClient.CoreV1().ConfigMaps("test-namespace").Get(ctx, pkgKubernetes.TLSCABundleConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	verifyConfigMapData(t, configMap.Data, []string{"First CA"})

	err = CreateTLSCABundleConfigMapFromCerts(ctx, []*x509.Certificate{cert2}, k8sClient)
	require.NoError(t, err)

	// Verify that creating the ConfigMap twice updates it instead of failing
	configMap, err = k8sClient.CoreV1().ConfigMaps("test-namespace").Get(ctx, pkgKubernetes.TLSCABundleConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	verifyConfigMapData(t, configMap.Data, []string{"Second CA"})
}

func verifyConfigMapData(t *testing.T, data map[string]string, expectedNames []string) {
	caBundle, exists := data[pkgKubernetes.TLSCABundleKey]
	assert.True(t, exists, "CA bundle should be present in ConfigMap data")
	assert.NotEmpty(t, caBundle, "CA bundle should not be empty")

	assert.Contains(t, caBundle, "-----BEGIN CERTIFICATE-----", "CA bundle should contain PEM certificates")
	assert.Contains(t, caBundle, "-----END CERTIFICATE-----", "CA bundle should contain PEM certificates")

	parsedCerts, err := x509utils.ConvertPEMTox509Certs([]byte(caBundle))
	require.NoError(t, err)
	require.Len(t, parsedCerts, len(expectedNames), "Should contain exactly %d certificates", len(expectedNames))

	for _, expectedName := range expectedNames {
		found := false
		for _, cert := range parsedCerts {
			if cert.Subject.CommonName == expectedName {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected certificate with name %s not found", expectedName)
	}
}

func createTestCertificate(t *testing.T, commonName string) *x509.Certificate {
	tlsCert := testutils.IssueSelfSignedCert(t, commonName)
	return tlsCert.Leaf
}
