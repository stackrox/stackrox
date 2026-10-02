// Package vmhelpers provides helpers shared by the VM-scanning e2e suite and
// its unit tests: loading VM scan configuration, KubeVirt/VSOCK cluster
// preflight checks, compliance metrics setup, and Central/roxagent
// interactions.
package vmhelpers

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
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"golang.org/x/crypto/ssh"
)

const (
	// Hardcoded because registerDurationSetting in pkg/env is unexported.
	// This is acceptable since adjusting these values requires a code change anyway.
	defaultScanTimeout      = 20 * time.Minute
	defaultScanPollInterval = 10 * time.Second
	defaultDeleteTimeout    = 5 * time.Minute
	defaultGuestUser        = "cloud-user"
)

var (
	vmScanNamespacePrefix = env.RegisterSetting("VM_SCAN_NAMESPACE_PREFIX", env.WithDefault("vm-scan-e2e"))
	vmScanSkipCleanup     = env.RegisterBooleanSetting("VM_SCAN_SKIP_CLEANUP", false)

	// repo2CPEURL is empty so Quadlet omits --repo-cpe-url and the agent
	// stays Sensor-managed. Set ROXAGENT_REPO2CPE_URL to force URL-managed.
	repo2CPEURL = env.RegisterSetting("ROXAGENT_REPO2CPE_URL")
)

// VMSpec describes a VM to provision: container-disk image and guest SSH user.
type VMSpec struct {
	Name      string
	Image     string
	GuestUser string
}

// VMScanConfig contains the VM scanning test configuration derived from the environment.
type VMScanConfig struct {
	Images              []string // container-disk images (from VM_IMAGES, comma-separated)
	GuestUsers          []string // per-image SSH users (from VM_USERS, comma-separated; shorter lists are padded with defaultGuestUser)
	VirtctlPath         string
	RoxagentImage       string
	Repo2CPEURL         string
	SSHPrivateKey       string // PEM-encoded private key content (not a file path)
	SSHPublicKey        string // OpenSSH authorized_keys line (not a file path)
	NamespacePrefix     string
	ScanTimeout         time.Duration
	ScanPollInterval    time.Duration
	DeleteTimeout       time.Duration
	SkipCleanup         bool
	ImagePullSecretPath string // Path to docker config JSON for the namespace imagePullSecret
	PodmanAuthFilePath  string // Path to containers-auth JSON for guest Quadlet pulls
}

// LoadVMScanConfig reads the VM scanning configuration from the environment.
func LoadVMScanConfig() (*VMScanConfig, error) {
	cfg := &VMScanConfig{}

	var err error
	imagesRaw := strings.TrimSpace(os.Getenv("VM_IMAGES"))
	if imagesRaw == "" {
		return nil, errors.New("VM_IMAGES is required (comma-separated list of container-disk image references)")
	}
	for img := range strings.SplitSeq(imagesRaw, ",") {
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
		for u := range strings.SplitSeq(usersRaw, ",") {
			cfg.GuestUsers = append(cfg.GuestUsers, strings.TrimSpace(u))
		}
	}
	if cfg.VirtctlPath, err = DiscoverVirtctlPath(); err != nil {
		return nil, err
	}
	if cfg.RoxagentImage, err = discoverRoxagentImage(); err != nil {
		return nil, err
	}

	cfg.Repo2CPEURL = repo2CPEURL.Setting()

	cfg.SSHPrivateKey = os.Getenv("VM_SSH_PRIVATE_KEY")
	cfg.SSHPublicKey = strings.TrimSpace(os.Getenv("VM_SSH_PUBLIC_KEY"))
	switch {
	case strings.TrimSpace(cfg.SSHPrivateKey) == "" && cfg.SSHPublicKey == "":
		priv, pub, genErr := GenerateEphemeralSSHKeypair()
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
	cfg.ScanPollInterval = defaultScanPollInterval
	cfg.DeleteTimeout = defaultDeleteTimeout
	cfg.SkipCleanup = vmScanSkipCleanup.BooleanSetting()
	cfg.ImagePullSecretPath = strings.TrimSpace(os.Getenv("VM_IMAGE_PULL_SECRET_PATH"))
	cfg.PodmanAuthFilePath = strings.TrimSpace(os.Getenv("VM_PODMAN_AUTH_FILE"))

	return cfg, nil
}

// VMSpecs builds the VM specification list from the parsed images and guest
// users. VM names are generated as vm-0, vm-1, etc.
func (c *VMScanConfig) VMSpecs() []VMSpec {
	specs := make([]VMSpec, len(c.Images))
	for i, img := range c.Images {
		user := defaultGuestUser
		if i < len(c.GuestUsers) && c.GuestUsers[i] != "" {
			user = c.GuestUsers[i]
		}
		specs[i] = VMSpec{
			Name:      fmt.Sprintf("vm-%d", i),
			Image:     img,
			GuestUser: user,
		}
	}
	return specs
}

// DiscoverVirtctlPath returns the VIRTCTL_PATH env var if set, otherwise searches $PATH.
func DiscoverVirtctlPath() (string, error) {
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

// discoverRoxagentImage returns ROXAGENT_IMAGE, else MAIN_IMAGE, else
// <repo>:MAIN_IMAGE_TAG. Repo is MAIN_IMAGE_REPO, else DEFAULT_IMAGE_REGISTRY/main,
// else the branding default, so CI can pass only MAIN_IMAGE_TAG.
func discoverRoxagentImage() (string, error) {
	if v := strings.TrimSpace(os.Getenv("ROXAGENT_IMAGE")); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv("MAIN_IMAGE")); v != "" {
		return v, nil
	}
	tag := strings.TrimSpace(os.Getenv("MAIN_IMAGE_TAG"))
	if tag == "" {
		return "", errors.New("ROXAGENT_IMAGE or MAIN_IMAGE is required (or MAIN_IMAGE_TAG)")
	}
	return defaultMainImageRepo() + ":" + tag, nil
}

// defaultMainImageRepo matches make/env.mk: RHACS_BRANDING uses quay.io/rhacs-eng,
// otherwise quay.io/stackrox-io.
func defaultMainImageRepo() string {
	if v := strings.TrimSpace(os.Getenv("MAIN_IMAGE_REPO")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("DEFAULT_IMAGE_REGISTRY")); v != "" {
		return v + "/main"
	}
	if os.Getenv("ROX_PRODUCT_BRANDING") == "RHACS_BRANDING" {
		return "quay.io/rhacs-eng/main"
	}
	return "quay.io/stackrox-io/main"
}

// repoRoot returns the repository root by walking up from this source file.
func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return repoRootFrom(file)
}

func repoRootFrom(file string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// GenerateEphemeralSSHKeypair creates a one-time ed25519 keypair and returns
// the PEM-encoded private key and the OpenSSH authorized_keys public key line.
func GenerateEphemeralSSHKeypair() (privateKeyPEM string, publicKeyAuthorized string, err error) {
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
