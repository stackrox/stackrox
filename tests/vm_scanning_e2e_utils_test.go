//go:build test_e2e || test_e2e_vm

package tests

import (
	"encoding/pem"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

func mustLoadVMScanConfig(t *testing.T) *vmScanConfig {
	t.Helper()
	cfg, err := loadVMScanConfig()
	require.NoError(t, err, "loadVMScanConfig")
	return cfg
}

func mustCreateDynamicClient(t *testing.T, restCfg *rest.Config) dynamic.Interface {
	t.Helper()
	c, err := dynamic.NewForConfig(restCfg)
	require.NoError(t, err, "dynamic.NewForConfig")
	return c
}

// mustResolveSSHIdentityFile writes the PEM-encoded private key content to a temporary file
// with 0600 permissions and returns the path, suitable for virtctl --identity-file.
func mustResolveSSHIdentityFile(t *testing.T, cfg *vmScanConfig) string {
	t.Helper()
	content := cfg.SSHPrivateKey
	trimmed := strings.TrimSpace(content)
	require.NotEmpty(t, trimmed, "SSH private key content is empty")
	block, _ := pem.Decode([]byte(trimmed))
	require.NotNil(t, block, "VM_SSH_PRIVATE_KEY must contain complete PEM-encoded key content, not a file path")

	// OpenSSH requires a trailing newline after the END marker.
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	f, err := os.CreateTemp(t.TempDir(), "vm-scan-ssh-*")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(f.Name(), 0o600))
	return f.Name()
}
