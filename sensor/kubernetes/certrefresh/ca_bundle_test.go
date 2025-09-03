package certrefresh

import (
	"bytes"
	"context"
	"crypto/x509"
	"testing"

	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/x509utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestConvertCertsToPEM(t *testing.T) {
	testCases := []struct {
		name       string
		certs      []*x509.Certificate
		shouldFail bool
	}{
		{
			name:       "empty input",
			certs:      []*x509.Certificate{},
			shouldFail: true,
		},
		{
			name:  "single certificate",
			certs: []*x509.Certificate{createTestCertificate(t, "Single CA")},
		},
		{
			name:  "two certificates",
			certs: []*x509.Certificate{createTestCertificate(t, "Primary CA"), createTestCertificate(t, "Secondary CA")},
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
			require.Len(t, parsedCerts, len(tc.certs))

			// Verify certificate data integrity
			for i, expectedCert := range tc.certs {
				found := false
				for _, parsedCert := range parsedCerts {
					if bytes.Equal(expectedCert.Raw, parsedCert.Raw) {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected certificate #%d (CN: %s) not found after PEM conversion", i, expectedCert.Subject.CommonName)
			}
		})
	}
}

func TestCreateTLSCABundleConfigMap(t *testing.T) {
	testCases := []struct {
		name       string
		certCount  int
		shouldFail bool
	}{
		{
			name:       "empty certificates",
			certCount:  0,
			shouldFail: true,
		},
		{
			name:      "single certificate",
			certCount: 1,
		},
		{
			name:      "multiple certificates",
			certCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("POD_NAMESPACE", "test-namespace")
			k8sClient := fake.NewSimpleClientset()

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

					trueVar := true
					ownerRef := &metav1.OwnerReference{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "test-deployment",
						UID:        "test-deployment-uid",
						Controller: &trueVar,
					}

					var err error
					switch testedFunc {
					case "from_certs":
						err = CreateTLSCABundleConfigMapFromCerts(ctx, certs, k8sClient.CoreV1().ConfigMaps("test-namespace"), ownerRef)
					case "from_pem":
						var pemData []byte
						if tc.certCount == 0 {
							pemData = []byte{}
						} else {
							pemData, err = convertCertsToPEM(certs)
							require.NoError(t, err)
						}
						err = CreateTLSCABundleConfigMapFromPEM(ctx, pemData, k8sClient.CoreV1().ConfigMaps("test-namespace"), ownerRef)
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

					verifyConfigMapData(t, configMap.Data, certs)
					assert.Equal(t, pkgKubernetes.TLSCABundleConfigMapName, configMap.Name)
					assert.Equal(t, "test-namespace", configMap.Namespace)
					assert.Contains(t, configMap.Annotations, tlsCABundleAnnotationKey)
					assert.Equal(t, labels.ManagedBySensor, configMap.Labels[labels.ManagedByLabelKey])
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

	k8sClient := fake.NewSimpleClientset()

	ctx := context.Background()
	cert1 := createTestCertificate(t, "First CA")
	cert2 := createTestCertificate(t, "Second CA")

	// Create a fake owner reference for the tests
	trueVar := true
	ownerRef := &metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
		Name:       "test-deployment",
		UID:        "test-deployment-uid",
		Controller: &trueVar,
	}

	err := CreateTLSCABundleConfigMapFromCerts(ctx, []*x509.Certificate{cert1}, k8sClient.CoreV1().ConfigMaps("test-namespace"), ownerRef)
	require.NoError(t, err)

	configMap, err := k8sClient.CoreV1().ConfigMaps("test-namespace").Get(ctx, pkgKubernetes.TLSCABundleConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	verifyConfigMapData(t, configMap.Data, []*x509.Certificate{cert1})

	err = CreateTLSCABundleConfigMapFromCerts(ctx, []*x509.Certificate{cert2}, k8sClient.CoreV1().ConfigMaps("test-namespace"), ownerRef)
	require.NoError(t, err)

	// Verify that creating the ConfigMap twice updates it instead of failing
	configMap, err = k8sClient.CoreV1().ConfigMaps("test-namespace").Get(ctx, pkgKubernetes.TLSCABundleConfigMapName, metav1.GetOptions{})
	require.NoError(t, err)
	verifyConfigMapData(t, configMap.Data, []*x509.Certificate{cert2})
}

func verifyConfigMapData(t *testing.T, data map[string]string, expectedCerts []*x509.Certificate) {
	caBundle, exists := data[pkgKubernetes.TLSCABundleKey]
	assert.True(t, exists, "CA bundle should be present in ConfigMap data")
	assert.NotEmpty(t, caBundle, "CA bundle should not be empty")

	assert.Contains(t, caBundle, "-----BEGIN CERTIFICATE-----", "CA bundle should contain PEM certificates")
	assert.Contains(t, caBundle, "-----END CERTIFICATE-----", "CA bundle should contain PEM certificates")

	parsedCerts, err := x509utils.ConvertPEMTox509Certs([]byte(caBundle))
	require.NoError(t, err)
	require.Len(t, parsedCerts, len(expectedCerts), "Should contain exactly %d certificates", len(expectedCerts))

	for i, expectedCert := range expectedCerts {
		found := false
		for _, parsedCert := range parsedCerts {
			if bytes.Equal(expectedCert.Raw, parsedCert.Raw) {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected certificate #%d (CN: %s) not found in ConfigMap", i, expectedCert.Subject.CommonName)
	}
}

func createTestCertificate(t *testing.T, commonName string) *x509.Certificate {
	tlsCert := testutils.IssueSelfSignedCert(t, commonName)
	return tlsCert.Leaf
}
