//go:build test_e2e

package tests

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Defaults for environment variables that can be self-discovered or have sensible values.
const (
	// defaultNamespacePrefix is the prefix for the namespace to provision the VMs in.
	defaultNamespacePrefix = "vm-scan-e2e"
	// defaultScanTimeout is the timeout for the scan to complete.
	defaultScanTimeout = 20 * time.Minute
	// defaultDeleteTimeout is the timeout for the delete to complete.
	defaultDeleteTimeout = 5 * time.Minute
	// defaultGuestUser is the user to use for the guest.
	defaultGuestUser = "cloud-user"
)

// vmSpec describes a VM to provision: container-disk image and guest SSH user.
type vmSpec struct {
	Name      string
	Image     string
	GuestUser string
}

type vmScanConfig struct {
	Images              []string // container-disk images (from VM_IMAGES, comma-separated)
	GuestUsers          []string // per-image SSH users (from VM_USERS, comma-separated; shorter lists are padded with defaultGuestUser)
	VirtctlPath         string
	SSHPrivateKey       string // PEM-encoded private key content (not a file path)
	SSHPublicKey        string // OpenSSH authorized_keys line (not a file path)
	NamespacePrefix     string
	ScanTimeout         time.Duration
	DeleteTimeout       time.Duration
	SkipCleanup         bool
	ImagePullSecretPath string // Path to docker config JSON for private registries
}

func loadVMScanConfig() (*vmScanConfig, error) {
	cfg := &vmScanConfig{}

	var err error
	imagesRaw := strings.TrimSpace(os.Getenv("VM_IMAGES"))
	if imagesRaw == "" {
		return nil, errors.New("VM_IMAGES is required (comma-separated list of container-disk image references)")
	}
	for _, img := range strings.Split(imagesRaw, ",") {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}
		cfg.Images = append(cfg.Images, img)
	}
	if len(cfg.Images) == 0 {
		return nil, errors.New("VM_IMAGES must contain at least one non-empty image reference")
	}

	if usersRaw := strings.TrimSpace(os.Getenv("VM_USERS")); usersRaw != "" {
		for _, u := range strings.Split(usersRaw, ",") {
			cfg.GuestUsers = append(cfg.GuestUsers, strings.TrimSpace(u))
		}
	}
	for len(cfg.GuestUsers) < len(cfg.Images) {
		cfg.GuestUsers = append(cfg.GuestUsers, defaultGuestUser)
	}

	if cfg.VirtctlPath, err = discoverVirtctlPath(); err != nil {
		return nil, err
	}

	cfg.SSHPrivateKey = os.Getenv("VM_SSH_PRIVATE_KEY")
	cfg.SSHPublicKey = strings.TrimSpace(os.Getenv("VM_SSH_PUBLIC_KEY"))
	switch {
	case strings.TrimSpace(cfg.SSHPrivateKey) == "" && cfg.SSHPublicKey == "":
		priv, pub, genErr := generateEphemeralSSHKeypair()
		if genErr != nil {
			return nil, fmt.Errorf("VM_SSH_PRIVATE_KEY/VM_SSH_PUBLIC_KEY not set and ephemeral key generation failed: %w", genErr)
		}
		cfg.SSHPrivateKey = priv
		cfg.SSHPublicKey = pub
	case strings.TrimSpace(cfg.SSHPrivateKey) == "":
		return nil, errors.New("VM_SSH_PUBLIC_KEY is set but VM_SSH_PRIVATE_KEY is missing; provide both or neither")
	case cfg.SSHPublicKey == "":
		return nil, errors.New("VM_SSH_PRIVATE_KEY is set but VM_SSH_PUBLIC_KEY is missing; provide both or neither")
	}
	trimmedKey := strings.TrimSpace(cfg.SSHPrivateKey)
	if !strings.HasPrefix(trimmedKey, "-----BEGIN") || !strings.Contains(trimmedKey, "-----END") {
		return nil, errors.New("VM_SSH_PRIVATE_KEY must contain complete PEM-encoded key content, not a file path")
	}

	cfg.NamespacePrefix = envOrDefault("VM_SCAN_NAMESPACE_PREFIX", defaultNamespacePrefix)
	if cfg.ScanTimeout, err = parseEnvDurationOrDefault("VM_SCAN_TIMEOUT", defaultScanTimeout); err != nil {
		return nil, err
	}
	if cfg.DeleteTimeout, err = parseEnvDurationOrDefault("VM_DELETE_TIMEOUT", defaultDeleteTimeout); err != nil {
		return nil, err
	}

	cfg.SkipCleanup = envTruthy("VM_SCAN_SKIP_CLEANUP")
	cfg.ImagePullSecretPath = strings.TrimSpace(os.Getenv("VM_IMAGE_PULL_SECRET_PATH"))

	return cfg, nil
}

// vmSpecs builds the VM specification list from the parsed images and guest
// users. VM names are generated as vm-0, vm-1, etc.
func (c *vmScanConfig) vmSpecs() []vmSpec {
	specs := make([]vmSpec, len(c.Images))
	for i, img := range c.Images {
		user := defaultGuestUser
		if i < len(c.GuestUsers) && c.GuestUsers[i] != "" {
			user = c.GuestUsers[i]
		}
		specs[i] = vmSpec{
			Name:      fmt.Sprintf("vm-%d", i),
			Image:     img,
			GuestUser: user,
		}
	}
	return specs
}

// discoverVirtctlPath returns the VIRTCTL_PATH env var if set, otherwise searches $PATH.
func discoverVirtctlPath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("VIRTCTL_PATH")); v != "" {
		info, err := os.Stat(v)
		if err != nil {
			return "", fmt.Errorf("VIRTCTL_PATH %q is not accessible: %w", v, err)
		}
		if !info.Mode().IsRegular() || (info.Mode()&0o111) == 0 {
			return "", fmt.Errorf("VIRTCTL_PATH %q is not an executable file", v)
		}
		return v, nil
	}
	p, err := exec.LookPath("virtctl")
	if err != nil {
		return "", fmt.Errorf("VIRTCTL_PATH not set and virtctl not found on $PATH: %w", err)
	}
	return p, nil
}

// generateEphemeralSSHKeypair creates a one-time ed25519 keypair and returns
// the PEM-encoded private key and the OpenSSH authorized_keys public key line.
func generateEphemeralSSHKeypair() (privateKeyPEM string, publicKeyAuthorized string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}
	privBytes, err := ssh.MarshalPrivateKey(priv, "stackrox-vm-scan-e2e-ephemeral")
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	pemData := pem.EncodeToMemory(privBytes)

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("convert public key: %w", err)
	}
	authorizedKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPub)))

	return string(pemData), authorizedKey, nil
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultVal
}

func parseEnvDurationOrDefault(key string, defaultVal time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s: invalid duration %q: %w", key, v, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("environment variable %s: duration must be > 0, got %q", key, v)
	}
	return d, nil
}

func TestLoadVMScanConfig_MissingImages(t *testing.T) {
	t.Setenv("VM_IMAGES", "")
	_, err := loadVMScanConfig()
	require.ErrorContains(t, err, "VM_IMAGES")
}

func TestLoadVMScanConfig_Defaults(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest,registry.example.com/rhel10:latest")
	t.Setenv("VM_USERS", "")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))
	for _, key := range []string{
		"VM_SCAN_NAMESPACE_PREFIX", "VM_SCAN_TIMEOUT", "VM_DELETE_TIMEOUT",
	} {
		t.Setenv(key, "")
	}
	cfg, err := loadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, []string{"registry.example.com/rhel9:latest", "registry.example.com/rhel10:latest"}, cfg.Images)
	require.Equal(t, []string{defaultGuestUser, defaultGuestUser}, cfg.GuestUsers)
	require.Equal(t, defaultNamespacePrefix, cfg.NamespacePrefix)
	require.Equal(t, defaultScanTimeout, cfg.ScanTimeout)
	require.Equal(t, defaultDeleteTimeout, cfg.DeleteTimeout)

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
	require.Equal(t, []string{"alice", defaultGuestUser, defaultGuestUser}, cfg.GuestUsers)
}

func TestLoadVMScanConfig_InvalidOptionalOverrides(t *testing.T) {
	t.Setenv("VM_IMAGES", "registry.example.com/rhel9:latest")
	t.Setenv("VIRTCTL_PATH", mustFindExecutable(t, "true"))

	testCases := []struct {
		name      string
		envKey    string
		envValue  string
		expectErr string
	}{
		{
			name:      "negative duration",
			envKey:    "VM_SCAN_TIMEOUT",
			envValue:  "-1s",
			expectErr: "VM_SCAN_TIMEOUT",
		},
		{
			name:      "zero duration",
			envKey:    "VM_DELETE_TIMEOUT",
			envValue:  "0s",
			expectErr: "VM_DELETE_TIMEOUT",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VM_SCAN_TIMEOUT", "")
			t.Setenv("VM_DELETE_TIMEOUT", "")
			t.Setenv(tc.envKey, tc.envValue)
			_, err := loadVMScanConfig()
			require.Error(t, err)
			require.ErrorContains(t, err, tc.expectErr)
		})
	}
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
			_, err := loadVMScanConfig()
			require.Error(t, err)
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

func mustFindExecutable(t *testing.T, name string) string {
	t.Helper()

	path, err := exec.LookPath(name)
	require.NoError(t, err)
	return path
}
