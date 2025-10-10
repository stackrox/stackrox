package certs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func createTestSecret(name, namespace string, data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

type testClient struct {
	k8s kubernetes.Interface
}

func (t *testClient) Kubernetes() kubernetes.Interface {
	return t.k8s
}

func (t *testClient) Dynamic() dynamic.Interface {
	return nil
}

func (t *testClient) OpenshiftApps() appVersioned.Interface {
	return nil
}

func (t *testClient) OpenshiftConfig() configVersioned.Interface {
	return nil
}

func (t *testClient) OpenshiftRoute() routeVersioned.Interface {
	return nil
}

func (t *testClient) OpenshiftOperator() operatorVersioned.Interface {
	return nil
}

func createTestClient(secrets ...*v1.Secret) client.Interface {
	fakeClient := fake.NewSimpleClientset()
	for _, secret := range secrets {
		_, _ = fakeClient.CoreV1().Secrets(secret.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	}
	return &testClient{
		k8s: fakeClient,
	}
}

func TestFetchCertificates(t *testing.T) {
	tests := map[string]struct {
		secrets           []*v1.Secret
		shouldSucceed     bool
		expectedCertFiles map[string]string // maps env var name -> expected filename
	}{
		"should fetch certificates from primary secret successfully": {
			secrets: []*v1.Secret{
				createTestSecret("tls-cert-sensor", DefaultNamespace, map[string][]byte{
					"ca.pem":   []byte("primary-ca-content"),
					"cert.pem": []byte("primary-cert-content"),
					"key.pem":  []byte("primary-key-content"),
				}),
			},
			shouldSucceed: true,
			expectedCertFiles: map[string]string{
				certEnvName:       "ca.pem",
				sensorCertEnvName: "cert.pem",
				sensorKeyEnvName:  "key.pem",
			},
		},
		"should fallback to legacy secret when primary is missing": {
			secrets: []*v1.Secret{
				createTestSecret("sensor-tls", DefaultNamespace, map[string][]byte{
					"ca.pem":          []byte("legacy-ca-content"),
					"sensor-cert.pem": []byte("legacy-cert-content"),
					"sensor-key.pem":  []byte("legacy-key-content"),
				}),
			},
			shouldSucceed: true,
			expectedCertFiles: map[string]string{
				certEnvName:       "ca.pem",
				sensorCertEnvName: "sensor-cert.pem",
				sensorKeyEnvName:  "sensor-key.pem",
			},
		},
		"should prefer primary secret when both exist": {
			secrets: []*v1.Secret{
				createTestSecret("tls-cert-sensor", DefaultNamespace, map[string][]byte{
					"ca.pem":   []byte("primary-ca-content"),
					"cert.pem": []byte("primary-cert-content"),
					"key.pem":  []byte("primary-key-content"),
				}),
				createTestSecret("sensor-tls", DefaultNamespace, map[string][]byte{
					"ca.pem":          []byte("legacy-ca-content"),
					"sensor-cert.pem": []byte("legacy-cert-content"),
					"sensor-key.pem":  []byte("legacy-key-content"),
				}),
			},
			shouldSucceed: true,
			expectedCertFiles: map[string]string{
				certEnvName:       "ca.pem",
				sensorCertEnvName: "cert.pem",
				sensorKeyEnvName:  "key.pem",
			},
		},
		"should fail when neither secret exists": {
			secrets:       []*v1.Secret{},
			shouldSucceed: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			k8sClient := createTestClient(tt.secrets...)

			capturedEnvVars := make(map[string]string)
			setEnvFunc := func(key, value string) error {
				capturedEnvVars[key] = value
				return nil
			}

			fetcher := NewCertificateFetcher(k8sClient,
				WithOutputDir(tmpDir),
				WithSetEnvFunc(setEnvFunc),
				WithHelmConfig("", "", ""),
				WithClusterName("", "", ""))

			err := fetcher.FetchCertificatesAndSetEnvironment()

			if tt.shouldSucceed {
				require.NoError(t, err)

				// Verify environment variables point to correct certificate files
				for envVar, expectedFilename := range tt.expectedCertFiles {
					expectedPath := filepath.Join(tmpDir, expectedFilename)
					assert.Equal(t, expectedPath, capturedEnvVars[envVar],
						"environment variable %s should point to %s", envVar, expectedFilename)

					// Verify certificate file was written
					_, err := os.Stat(expectedPath)
					assert.NoError(t, err, "certificate file %s should exist", expectedFilename)
				}
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to fetch certificates from any source")
			}
		})
	}
}

func TestFetchCertificates_CustomConfig(t *testing.T) {
	tests := map[string]struct {
		customConfigs []CertConfig
		secrets       []*v1.Secret
		shouldSucceed bool
	}{
		"should use single custom config instead of defaults": {
			customConfigs: []CertConfig{
				{
					SecretName: "custom-secret",
					CertNames: map[string]string{
						certEnvName:       "custom-ca.pem",
						sensorCertEnvName: "custom-cert.pem",
						sensorKeyEnvName:  "custom-key.pem",
					},
				},
			},
			secrets: []*v1.Secret{
				createTestSecret("custom-secret", DefaultNamespace, map[string][]byte{
					"custom-ca.pem":   []byte("custom-ca"),
					"custom-cert.pem": []byte("custom-cert"),
					"custom-key.pem":  []byte("custom-key"),
				}),
			},
			shouldSucceed: true,
		},
		"should try multiple custom configs in order": {
			customConfigs: []CertConfig{
				{
					SecretName: "missing-secret",
					CertNames: map[string]string{
						certEnvName: "ca.pem",
					},
				},
				{
					SecretName: "backup-secret",
					CertNames: map[string]string{
						certEnvName: "backup-ca.pem",
					},
				},
			},
			secrets: []*v1.Secret{
				createTestSecret("backup-secret", DefaultNamespace, map[string][]byte{
					"backup-ca.pem": []byte("backup-ca"),
				}),
			},
			shouldSucceed: true,
		},
		"should not fallback to defaults when custom config provided": {
			customConfigs: []CertConfig{
				{
					SecretName: "non-existent-secret",
					CertNames: map[string]string{
						certEnvName: "ca.pem",
					},
				},
			},
			secrets: []*v1.Secret{
				// Even though default secrets exist, they should not be used
				createTestSecret("tls-cert-sensor", DefaultNamespace, map[string][]byte{
					"ca.pem": []byte("default-ca"),
				}),
			},
			shouldSucceed: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			k8sClient := createTestClient(tt.secrets...)

			capturedEnvVars := make(map[string]string)
			setEnvFunc := func(key, value string) error {
				capturedEnvVars[key] = value
				return nil
			}

			fetcher := NewCertificateFetcher(k8sClient,
				WithOutputDir(tmpDir),
				WithSetEnvFunc(setEnvFunc),
				WithCertConfig(tt.customConfigs...),
				WithHelmConfig("", "", ""),
				WithClusterName("", "", ""))

			err := fetcher.FetchCertificatesAndSetEnvironment()

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to fetch certificates from any source")
			}
		})
	}
}

func TestFetchCertificates_MissingCertificateFile(t *testing.T) {
	tests := map[string]struct {
		secretData map[string][]byte
	}{
		"should fail when ca.pem is missing from primary secret": {
			secretData: map[string][]byte{
				"cert.pem": []byte("cert-content"),
				"key.pem":  []byte("key-content"),
			},
		},
		"should fail when cert.pem is missing from primary secret": {
			secretData: map[string][]byte{
				"ca.pem":  []byte("ca-content"),
				"key.pem": []byte("key-content"),
			},
		},
		"should fail when key.pem is missing from primary secret": {
			secretData: map[string][]byte{
				"ca.pem":   []byte("ca-content"),
				"cert.pem": []byte("cert-content"),
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			secret := createTestSecret("tls-cert-sensor", DefaultNamespace, tt.secretData)
			k8sClient := createTestClient(secret)

			fetcher := NewCertificateFetcher(k8sClient,
				WithOutputDir(tmpDir),
				WithHelmConfig("", "", ""),
				WithClusterName("", "", ""))

			err := fetcher.FetchCertificatesAndSetEnvironment()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found in the secret")
		})
	}
}

func TestFetchCertificates_NamespaceOption(t *testing.T) {
	tests := map[string]struct {
		namespace string
	}{
		"should fetch certificates from custom namespace": {
			namespace: "custom-namespace",
		},
		"should fetch certificates from default namespace when not specified": {
			namespace: DefaultNamespace,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			secret := createTestSecret("tls-cert-sensor", tt.namespace, map[string][]byte{
				"ca.pem":   []byte("ca-content"),
				"cert.pem": []byte("cert-content"),
				"key.pem":  []byte("key-content"),
			})
			k8sClient := createTestClient(secret)

			var fetcher *CertificateFetcher
			if tt.namespace == DefaultNamespace {
				fetcher = NewCertificateFetcher(k8sClient,
					WithOutputDir(tmpDir),
					WithHelmConfig("", "", ""),
					WithClusterName("", "", ""))
			} else {
				fetcher = NewCertificateFetcher(k8sClient,
					WithOutputDir(tmpDir),
					WithNamespace(tt.namespace),
					WithHelmConfig("", "", ""),
					WithClusterName("", "", ""))
			}

			err := fetcher.FetchCertificatesAndSetEnvironment()
			assert.NoError(t, err)
		})
	}
}
