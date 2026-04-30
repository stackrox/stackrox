//go:build test_e2e

package tests

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadVMScanConfig_MissingImages(t *testing.T) {
	t.Setenv("VM_IMAGES", "")
	cfg, err := loadVMScanConfig()
	require.ErrorContains(t, err, "VM_IMAGES")
	require.Nil(t, cfg)
}

func TestLoadVMScanConfig_Defaults(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest,registry.example.com/rhel10:latest")
	t.Setenv("VM_USERS", "")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	t.Setenv("VM_SCAN_NAMESPACE_PREFIX", "")
	cfg, err := loadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"registry.example.com/rhel9:latest", "registry.example.com/rhel10:latest"}, cfg.Images)
	require.Empty(t, cfg.GuestUsers, "no padding; vmSpecs() defaults per-image")
	require.Equal(t, "vm-scan-e2e", cfg.NamespacePrefix)
	require.Equal(t, 20*time.Minute, cfg.ScanTimeout)
	require.Equal(t, 5*time.Minute, cfg.DeleteTimeout)

	specs := cfg.vmSpecs()
	require.Len(t, specs, 2)
	require.Equal(t, "vm-0", specs[0].Name)
	require.Equal(t, "vm-1", specs[1].Name)
}

func TestLoadVMScanConfig_PartialUsers(t *testing.T) {
	t.Setenv("VM_IMAGES", "img-a,img-b,img-c")
	t.Setenv("VM_USERS", "alice")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	cfg, err := loadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"alice"}, cfg.GuestUsers, "only explicit users; vmSpecs() pads with default")
}

func TestLoadVMScanConfig_InvalidSSHKeyContent(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))

	tests := map[string]string{
		"should reject a file path":         "/home/user/.ssh/id_ed25519",
		"should reject truncated PEM":       "-----BEGIN CERTIFICATE-----\nAAAA", // notsecret
		"should reject arbitrary non-PEM":   "not-a-key-at-all",
		"should reject value with only END": "-----END OPENSSH PRIVATE KEY-----",
	}
	for name, badKey := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("VM_SSH_PRIVATE_KEY", badKey)
			t.Setenv("VM_SSH_PUBLIC_KEY", "ssh-ed25519 AAAA test@host")
			cfg, err := loadVMScanConfig()
			require.Error(t, err)
			require.Nil(t, cfg)
			require.ErrorContains(t, err, "VM_SSH_PRIVATE_KEY must contain complete PEM-encoded key content")
		})
	}
}

func TestDiscoverVirtctlPath_InvalidEnvOverride(t *testing.T) {
	t.Run("missing file should return error", func(t *testing.T) {
		missing := t.TempDir() + "/virtctl-does-not-exist"
		t.Setenv("VIRTCTL_PATH", missing)
		_, err := discoverVirtctlPath()
		require.ErrorContains(t, err, "is not accessible")
	})

	t.Run("directory should return error", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VIRTCTL_PATH", dir)
		_, err := discoverVirtctlPath()
		require.ErrorContains(t, err, "is not an executable file")
	})

	t.Run("non executable file should return error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := tmpDir + "/virtctl"
		err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600)
		require.NoError(t, err)

		t.Setenv("VIRTCTL_PATH", path)
		_, err = discoverVirtctlPath()
		require.ErrorContains(t, err, "is not an executable file")
	})
}

func TestGenerateEphemeralSSHKeypair(t *testing.T) {
	priv, pub, err := generateEphemeralSSHKeypair()
	require.NoError(t, err)
	require.Contains(t, priv, "-----BEGIN OPENSSH PRIVATE KEY-----") // notsecret
	require.Contains(t, pub, "ssh-ed25519 ")                         // notsecret
}
