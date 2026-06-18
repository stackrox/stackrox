package k8sutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

const testKubeconfig = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

func TestGetK8sInClusterConfigFallback(t *testing.T) {
	// Ensure in-cluster environment variables are not set so rest.InClusterConfig fails.
	t.Setenv("KUBERNETES_SERVICE_HOST", "")
	t.Setenv("KUBERNETES_SERVICE_PORT", "")

	kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0o600))

	cases := map[string]struct {
		allowFallback  string
		kubeconfig     string
		managedCentral string
		expectErr      error
		expectHost     string
	}{
		"fallback disabled by default even when KUBECONFIG is set": {
			allowFallback: "",
			kubeconfig:    kubeconfigPath,
			expectErr:     rest.ErrNotInCluster,
		},
		"fallback explicitly disabled": {
			allowFallback: "false",
			kubeconfig:    kubeconfigPath,
			expectErr:     rest.ErrNotInCluster,
		},
		"fallback enabled loads KUBECONFIG": {
			allowFallback: "true",
			kubeconfig:    kubeconfigPath,
			expectHost:    "https://127.0.0.1:6443",
		},
		"fallback enabled does not rewrite host when ManagedCentral is true": {
			allowFallback:  "true",
			kubeconfig:     kubeconfigPath,
			managedCentral: "true",
			expectHost:     "https://127.0.0.1:6443",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ROX_ALLOW_KUBECONFIG_FALLBACK", tc.allowFallback)
			t.Setenv("KUBECONFIG", tc.kubeconfig)
			t.Setenv("ROX_MANAGED_CENTRAL", tc.managedCentral)

			cfg, err := GetK8sInClusterConfig()
			if tc.expectErr != nil {
				assert.ErrorIs(t, err, tc.expectErr)
				assert.Nil(t, cfg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, tc.expectHost, cfg.Host)
			assert.Equal(t, clientContentType, cfg.ContentType)
			assert.NotNil(t, cfg.WarningHandler)
		})
	}
}
