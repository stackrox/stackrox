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

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

const (
	// Hardcoded because registerDurationSetting in pkg/env is unexported.
	// This is acceptable since adjusting these values requires a code change anyway.
	defaultScanTimeout   = 20 * time.Minute
	defaultDeleteTimeout = 5 * time.Minute
	defaultGuestUser     = "cloud-user"
)

var (
	vmScanNamespacePrefix = env.RegisterSetting("VM_SCAN_NAMESPACE_PREFIX", env.WithDefault("vm-scan-e2e"))
	vmScanSkipCleanup     = env.RegisterBooleanSetting("VM_SCAN_SKIP_CLEANUP", false)
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

	cfg.NamespacePrefix = vmScanNamespacePrefix.Setting()
	cfg.ScanTimeout = defaultScanTimeout
	cfg.DeleteTimeout = defaultDeleteTimeout
	cfg.SkipCleanup = vmScanSkipCleanup.BooleanSetting()
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

func mustFindExecutable(t *testing.T, name string) string {
	t.Helper()

	path, err := exec.LookPath(name)
	require.NoError(t, err)
	return path
}
