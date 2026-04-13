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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// Defaults for environment variables that can be self-discovered or have sensible values.
const (
	defaultRepo2CPEPrimaryURL  = "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
	defaultRepo2CPEFallbackURL = "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
	defaultRepo2CPEAttempts    = 3
	defaultNamespacePrefix     = "vm-scan-e2e"
	defaultScanTimeout         = 20 * time.Minute
	defaultScanPollInterval    = 10 * time.Second
	defaultDeleteTimeout       = 5 * time.Minute
	defaultGuestUser           = "cloud-user"
)

type vmScanConfig struct {
	ImageRHEL9              string
	ImageRHEL10             string
	GuestUserRHEL9          string
	GuestUserRHEL10         string
	VirtctlPath             string
	RoxagentBinaryPath      string
	Repo2CPEPrimaryURL      string
	Repo2CPEFallbackURL     string
	Repo2CPEPrimaryAttempts int
	SSHPrivateKey           string // PEM-encoded private key content (not a file path)
	SSHPublicKey            string // OpenSSH authorized_keys line (not a file path)
	NamespacePrefix         string
	ScanTimeout             time.Duration
	ScanPollInterval        time.Duration
	DeleteTimeout           time.Duration
	RequireActivation       bool
	ActivationOrg           string
	ActivationKey           string
	ActivationEndpoint      string
	SkipCleanup             bool
	ImagePullSecretPath     string // Path to docker config JSON for private registries
}

func loadVMScanConfig() (*vmScanConfig, error) {
	cfg := &vmScanConfig{}

	var err error
	if cfg.ImageRHEL9, err = requireEnv("VM_IMAGE_RHEL9"); err != nil {
		return nil, err
	}
	if cfg.ImageRHEL10, err = requireEnv("VM_IMAGE_RHEL10"); err != nil {
		return nil, err
	}

	cfg.GuestUserRHEL9 = envOrDefault("VM_GUEST_USER_RHEL9", defaultGuestUser)
	cfg.GuestUserRHEL10 = envOrDefault("VM_GUEST_USER_RHEL10", defaultGuestUser)

	if cfg.VirtctlPath, err = discoverVirtctlPath(); err != nil {
		return nil, err
	}
	if cfg.RoxagentBinaryPath, err = discoverRoxagentBinaryPath(); err != nil {
		return nil, err
	}

	cfg.Repo2CPEPrimaryURL = envOrDefault("ROXAGENT_REPO2CPE_PRIMARY_URL", defaultRepo2CPEPrimaryURL)
	cfg.Repo2CPEFallbackURL = envOrDefault("ROXAGENT_REPO2CPE_FALLBACK_URL", defaultRepo2CPEFallbackURL)
	cfg.Repo2CPEPrimaryAttempts = envIntOrDefault("ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS", defaultRepo2CPEAttempts)

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

	cfg.NamespacePrefix = envOrDefault("VM_SCAN_NAMESPACE_PREFIX", defaultNamespacePrefix)
	cfg.ScanTimeout = envDurationOrDefault("VM_SCAN_TIMEOUT", defaultScanTimeout)
	cfg.ScanPollInterval = envDurationOrDefault("VM_SCAN_POLL_INTERVAL", defaultScanPollInterval)
	cfg.DeleteTimeout = envDurationOrDefault("VM_DELETE_TIMEOUT", defaultDeleteTimeout)

	cfg.ActivationOrg = strings.TrimSpace(os.Getenv("RHEL_ACTIVATION_ORG"))
	cfg.ActivationKey = strings.TrimSpace(os.Getenv("RHEL_ACTIVATION_KEY"))
	cfg.ActivationEndpoint = strings.TrimSpace(os.Getenv("RHEL_ACTIVATION_ENDPOINT"))

	// Derive RequireActivation: explicit env takes precedence, otherwise infer from credentials.
	if v := strings.TrimSpace(os.Getenv("VM_SCAN_REQUIRE_ACTIVATION")); v != "" {
		cfg.RequireActivation = envTruthy("VM_SCAN_REQUIRE_ACTIVATION")
	} else {
		cfg.RequireActivation = cfg.ActivationOrg != "" && cfg.ActivationKey != ""
	}
	if cfg.RequireActivation {
		if cfg.ActivationOrg == "" {
			return nil, errors.New("activation is required but RHEL_ACTIVATION_ORG is not set")
		}
		if cfg.ActivationKey == "" {
			return nil, errors.New("activation is required but RHEL_ACTIVATION_KEY is not set")
		}
	}

	cfg.SkipCleanup = envTruthy("VM_SCAN_SKIP_CLEANUP")
	cfg.ImagePullSecretPath = strings.TrimSpace(os.Getenv("VM_IMAGE_PULL_SECRET_PATH"))

	return cfg, nil
}

// discoverVirtctlPath returns the VIRTCTL_PATH env var if set, otherwise searches $PATH.
func discoverVirtctlPath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("VIRTCTL_PATH")); v != "" {
		return v, nil
	}
	p, err := exec.LookPath("virtctl")
	if err != nil {
		return "", fmt.Errorf("VIRTCTL_PATH not set and virtctl not found on $PATH: %w", err)
	}
	return p, nil
}

// discoverRoxagentBinaryPath returns the ROXAGENT_BINARY_PATH env var if set,
// otherwise probes the standard build output path relative to the repository root.
func discoverRoxagentBinaryPath() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ROXAGENT_BINARY_PATH")); v != "" {
		return v, nil
	}
	root := repoRoot()
	candidate := filepath.Join(root, "bin", "linux_amd64", "roxagent")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", fmt.Errorf("ROXAGENT_BINARY_PATH not set and %s does not exist; run 'make roxagent_linux-amd64'", candidate)
}

// repoRoot returns the repository root by walking up from this source file.
func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
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

func envIntOrDefault(key string, defaultVal int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func envDurationOrDefault(key string, defaultVal time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}

func requireEnv(key string) (string, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

func TestLoadVMScanConfig_MissingRequired(t *testing.T) {
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "false")
	t.Setenv("VM_IMAGE_RHEL9", "")
	_, err := loadVMScanConfig()
	require.ErrorContains(t, err, "VM_IMAGE_RHEL9")
}

func TestLoadVMScanConfig_RequiresActivationInputsWhenEnabled(t *testing.T) {
	t.Setenv("VM_IMAGE_RHEL9", "registry.example.com/rhel9:latest")
	t.Setenv("VM_IMAGE_RHEL10", "registry.example.com/rhel10:latest")
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "true")
	t.Setenv("RHEL_ACTIVATION_ORG", "")
	_, err := loadVMScanConfig()
	require.ErrorContains(t, err, "RHEL_ACTIVATION_ORG")
}

func TestLoadVMScanConfig_RequiresActivationKeyWhenEnabled(t *testing.T) {
	t.Setenv("VM_IMAGE_RHEL9", "registry.example.com/rhel9:latest")
	t.Setenv("VM_IMAGE_RHEL10", "registry.example.com/rhel10:latest")
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "true")
	t.Setenv("RHEL_ACTIVATION_ORG", "org-example")
	t.Setenv("RHEL_ACTIVATION_KEY", "")
	_, err := loadVMScanConfig()
	require.ErrorContains(t, err, "RHEL_ACTIVATION_KEY")
}

func TestLoadVMScanConfig_RequiresActivationOrgWhenTruthyNonTrue(t *testing.T) {
	t.Setenv("VM_IMAGE_RHEL9", "registry.example.com/rhel9:latest")
	t.Setenv("VM_IMAGE_RHEL10", "registry.example.com/rhel10:latest")
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "YES")
	t.Setenv("RHEL_ACTIVATION_ORG", "")
	_, err := loadVMScanConfig()
	require.ErrorContains(t, err, "RHEL_ACTIVATION_ORG")
}

func TestLoadVMScanConfig_DerivesActivationFromCredentials(t *testing.T) {
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "")
	t.Setenv("RHEL_ACTIVATION_ORG", "org-123")
	t.Setenv("RHEL_ACTIVATION_KEY", "key-456")
	t.Setenv("VM_IMAGE_RHEL9", "registry.example.com/rhel9:latest")
	t.Setenv("VM_IMAGE_RHEL10", "registry.example.com/rhel10:latest")
	cfg, err := loadVMScanConfig()
	require.NoError(t, err)
	require.True(t, cfg.RequireActivation, "should derive activation from credentials")
}

func TestLoadVMScanConfig_NoActivationWithoutCredentials(t *testing.T) {
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "")
	t.Setenv("RHEL_ACTIVATION_ORG", "")
	t.Setenv("RHEL_ACTIVATION_KEY", "")
	t.Setenv("VM_IMAGE_RHEL9", "registry.example.com/rhel9:latest")
	t.Setenv("VM_IMAGE_RHEL10", "registry.example.com/rhel10:latest")
	cfg, err := loadVMScanConfig()
	require.NoError(t, err)
	require.False(t, cfg.RequireActivation, "should not require activation without credentials")
}

func TestLoadVMScanConfig_Defaults(t *testing.T) {
	t.Setenv("VM_SCAN_REQUIRE_ACTIVATION", "")
	t.Setenv("RHEL_ACTIVATION_ORG", "")
	t.Setenv("RHEL_ACTIVATION_KEY", "")
	t.Setenv("VM_IMAGE_RHEL9", "registry.example.com/rhel9:latest")
	t.Setenv("VM_IMAGE_RHEL10", "registry.example.com/rhel10:latest")
	// Clear all optional vars to exercise defaults.
	for _, key := range []string{
		"VM_GUEST_USER_RHEL9", "VM_GUEST_USER_RHEL10",
		"VM_SCAN_NAMESPACE_PREFIX", "VM_SCAN_TIMEOUT", "VM_SCAN_POLL_INTERVAL", "VM_DELETE_TIMEOUT",
		"ROXAGENT_REPO2CPE_PRIMARY_URL", "ROXAGENT_REPO2CPE_FALLBACK_URL", "ROXAGENT_REPO2CPE_PRIMARY_ATTEMPTS",
	} {
		t.Setenv(key, "")
	}
	cfg, err := loadVMScanConfig()
	require.NoError(t, err)
	require.Equal(t, defaultGuestUser, cfg.GuestUserRHEL9)
	require.Equal(t, defaultGuestUser, cfg.GuestUserRHEL10)
	require.Equal(t, defaultNamespacePrefix, cfg.NamespacePrefix)
	require.Equal(t, defaultScanTimeout, cfg.ScanTimeout)
	require.Equal(t, defaultScanPollInterval, cfg.ScanPollInterval)
	require.Equal(t, defaultDeleteTimeout, cfg.DeleteTimeout)
	require.Equal(t, defaultRepo2CPEAttempts, cfg.Repo2CPEPrimaryAttempts)
}

func TestGenerateEphemeralSSHKeypair(t *testing.T) {
	priv, pub, err := generateEphemeralSSHKeypair()
	require.NoError(t, err)
	require.Contains(t, priv, "-----BEGIN OPENSSH PRIVATE KEY-----") // notsecret
	require.Contains(t, pub, "ssh-ed25519 ")                         // notsecret
}
