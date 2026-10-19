//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mustFindExecutable resolves an executable on PATH for tests that need a known-good binary.
func mustFindExecutable(t *testing.T, name string) string {
	t.Helper()

	path, err := exec.LookPath(name)
	require.NoError(t, err)
	return path
}

func TestLoadVMScanConfig_MissingImages(t *testing.T) {
	t.Setenv("VM_IMAGES", "")
	cfg, err := LoadVMScanConfig()
	require.ErrorContains(t, err, "VM_IMAGES")
	require.Nil(t, cfg)
}

func TestLoadVMScanConfig_Defaults(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest,registry.example.com/rhel10:latest")
	t.Setenv("VM_USERS", "")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	t.Setenv("ROXAGENT_IMAGE", "")
	t.Setenv("MAIN_IMAGE", "quay.io/example/main:test")
	t.Setenv("VM_SCAN_NAMESPACE_PREFIX", "")
	t.Setenv("ROXAGENT_REPO2CPE_URL", "")
	cfg, err := LoadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"registry.example.com/rhel9:latest", "registry.example.com/rhel10:latest"}, cfg.Images)
	require.Empty(t, cfg.GuestUsers, "no padding; VMSpecs() defaults per-image")
	require.Empty(t, cfg.Repo2CPEURL, "empty URL selects Sensor-managed mapping updater")
	require.Equal(t, "vm-scan-e2e", cfg.NamespacePrefix)
	require.Equal(t, 20*time.Minute, cfg.ScanTimeout)
	require.Equal(t, 10*time.Second, cfg.ScanPollInterval)
	require.Equal(t, 5*time.Minute, cfg.DeleteTimeout)
	require.Equal(t, "quay.io/example/main:test", cfg.RoxagentImage)

	specs := cfg.VMSpecs()
	require.Len(t, specs, 2)
	require.Equal(t, "vm-0", specs[0].Name)
	require.Equal(t, "vm-1", specs[1].Name)
}

func TestLoadVMScanConfig_AuthPaths(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	t.Setenv("MAIN_IMAGE", "quay.io/example/main:test")
	t.Setenv("VM_IMAGE_PULL_SECRET_PATH", "/tmp/k8s-pull-secret.json")
	t.Setenv("VM_PODMAN_AUTH_FILE", "/tmp/podman-auth.json")
	cfg, err := LoadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, "/tmp/k8s-pull-secret.json", cfg.ImagePullSecretPath)
	require.Equal(t, "/tmp/podman-auth.json", cfg.PodmanAuthFilePath)
}

func TestLoadVMScanConfig_PartialUsers(t *testing.T) {
	t.Setenv("VM_IMAGES", "img-a,img-b,img-c")
	t.Setenv("VM_USERS", "alice")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	t.Setenv("MAIN_IMAGE", "quay.io/example/main:test")
	cfg, err := LoadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"alice"}, cfg.GuestUsers, "only explicit users; VMSpecs() pads with default")
}

func TestLoadVMScanConfig_InvalidSSHKeyContent(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	t.Setenv("MAIN_IMAGE", "quay.io/example/main:test")

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
			cfg, err := LoadVMScanConfig()
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
		_, err := DiscoverVirtctlPath()
		require.ErrorContains(t, err, "is not accessible")
	})

	t.Run("directory should return error", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("VIRTCTL_PATH", dir)
		_, err := DiscoverVirtctlPath()
		require.ErrorContains(t, err, "is not an executable file")
	})

	t.Run("non executable file should return error", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := tmpDir + "/virtctl"
		err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o600)
		require.NoError(t, err)

		t.Setenv("VIRTCTL_PATH", path)
		_, err = DiscoverVirtctlPath()
		require.ErrorContains(t, err, "is not an executable file")
	})
}

func TestGenerateEphemeralSSHKeypair(t *testing.T) {
	priv, pub, err := GenerateEphemeralSSHKeypair()
	require.NoError(t, err)
	require.Contains(t, priv, "-----BEGIN OPENSSH PRIVATE KEY-----") // notsecret
	require.Contains(t, pub, "ssh-ed25519 ")                         // notsecret
}

func TestDiscoverRoxagentImage(t *testing.T) {
	tests := map[string]struct {
		roxagentImage string
		mainImage     string
		repo          string
		tag           string
		branding      string
		registry      string
		want          string
		errSubstring  string
	}{
		"should prefer ROXAGENT_IMAGE over MAIN_IMAGE": {
			roxagentImage: "quay.io/custom/main:dev",
			mainImage:     "quay.io/example/main:ignored",
			want:          "quay.io/custom/main:dev",
		},
		"should use MAIN_IMAGE": {
			mainImage: "quay.io/example/main:test",
			want:      "quay.io/example/main:test",
		},
		"should synthesize MAIN_IMAGE_REPO and MAIN_IMAGE_TAG": {
			repo: "quay.io/rhacs-eng/main",
			tag:  "4.12.x-123",
			want: "quay.io/rhacs-eng/main:4.12.x-123",
		},
		"should default rhacs repo from MAIN_IMAGE_TAG": {
			tag:      "4.12.x-784-g58e138e64c",
			branding: "RHACS_BRANDING",
			want:     "quay.io/rhacs-eng/main:4.12.x-784-g58e138e64c",
		},
		"should default opensource repo from MAIN_IMAGE_TAG": {
			tag:  "4.12.x-1",
			want: "quay.io/stackrox-io/main:4.12.x-1",
		},
		"should prefer DEFAULT_IMAGE_REGISTRY over branding": {
			tag:      "4.12.x-1",
			branding: "RHACS_BRANDING",
			registry: "quay.io/example",
			want:     "quay.io/example/main:4.12.x-1",
		},
		"should reject missing image configuration": {
			errSubstring: "ROXAGENT_IMAGE or MAIN_IMAGE",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("ROXAGENT_IMAGE", tc.roxagentImage)
			t.Setenv("MAIN_IMAGE", tc.mainImage)
			t.Setenv("MAIN_IMAGE_REPO", tc.repo)
			t.Setenv("MAIN_IMAGE_TAG", tc.tag)
			t.Setenv("ROX_PRODUCT_BRANDING", tc.branding)
			t.Setenv("DEFAULT_IMAGE_REGISTRY", tc.registry)
			got, err := discoverRoxagentImage()
			if tc.errSubstring != "" {
				require.ErrorContains(t, err, tc.errSubstring)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRepoRootFrom_VMHelpersFile(t *testing.T) {
	t.Parallel()

	got := repoRootFrom(filepath.Join("/repo", "tests", "vmhelpers", "vm_scan_config.go"))
	require.Equal(t, "/repo", got)
}
