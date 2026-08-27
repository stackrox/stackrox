//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOverlayQuadletContainer(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile(filepath.Join(repoRoot(), "compliance", "virtualmachines", "roxagent", "quadlet", "roxagent.container"))
	require.NoError(t, err)

	const (
		image = "quay.io/rhacs-eng/main:e2e-test"
		url   = "https://example.test/repository-to-cpe.json"
	)

	got, err := overlayQuadletContainer(src, image, url, "")
	require.NoError(t, err)
	out := string(got)
	require.Contains(t, out, "Image="+image)
	require.NotContains(t, out, "Image=quay.io/stackrox-io/main:latest")
	require.NotContains(t, out, "AuthFile=")
	require.NotContains(t, out, "REGISTRY_AUTH_FILE")
	require.Contains(t, out, "Exec=serve --host-path /host --rescan-interval "+E2ERescanInterval.String()+" --repo-cpe-url "+url)
	require.Contains(t, out, "Notify=true")
	require.Contains(t, out, "TimeoutStartSec=600")
}

func TestOverlayQuadletContainer_RegistryAuthFile(t *testing.T) {
	t.Parallel()

	src, err := os.ReadFile(filepath.Join(repoRoot(), "compliance", "virtualmachines", "roxagent", "quadlet", "roxagent.container"))
	require.NoError(t, err)

	got, err := overlayQuadletContainer(src, "quay.io/rhacs-eng/main:e2e-test", "https://example.test/repository-to-cpe.json", guestPodmanAuthPath)
	require.NoError(t, err)
	out := string(got)
	require.NotContains(t, out, "AuthFile=")
	container, rest, found := strings.Cut(out, "[Service]")
	require.True(t, found)
	require.NotContains(t, container, "REGISTRY_AUTH_FILE")
	require.Contains(t, rest, "Environment=REGISTRY_AUTH_FILE="+guestPodmanAuthPath)
}

func TestOverlayQuadletContainer_Errors(t *testing.T) {
	t.Parallel()

	src := []byte("[Container]\nImage=example:latest\nExec=serve\n")
	tests := map[string]struct {
		src          []byte
		image        string
		repo2cpeURL  string
		errSubstring string
	}{
		"should reject empty image": {
			src:          src,
			repo2cpeURL:  "https://example.test/mapping.json",
			errSubstring: "image is empty",
		},
		"should reject empty repo2cpe URL": {
			src:          src,
			image:        "example:latest",
			errSubstring: "repo2cpeURL is empty",
		},
		"should reject missing Image=": {
			src:          []byte("[Container]\nExec=serve\n"),
			image:        "example:latest",
			repo2cpeURL:  "https://example.test/mapping.json",
			errSubstring: "no Image= line",
		},
		"should reject missing Exec=": {
			src:          []byte("[Container]\nImage=example:latest\n"),
			image:        "example:latest",
			repo2cpeURL:  "https://example.test/mapping.json",
			errSubstring: "no Exec= line",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := overlayQuadletContainer(tc.src, tc.image, tc.repo2cpeURL, "")
			require.ErrorContains(t, err, tc.errSubstring)
		})
	}
}

func TestIsVsockUnavailableOutput(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		output string
		want   bool
	}{
		"vsock no such device": {
			output: "dial vsock host(2):818: connect: no such device",
			want:   true,
		},
		"dev vsock missing": {
			output: "open /dev/vsock: no such file or directory",
			want:   true,
		},
		"listen vsock missing": {
			output: "listening on VSOCK port 818: no such device",
			want:   true,
		},
		"podman device not found": {
			output: "error adding device /dev/vsock: not found",
			want:   true,
		},
		"non-vsock no such device": {
			output: "open /dev/does-not-exist: no such device",
			want:   false,
		},
		"other vsock error is retryable": {
			output: "dial vsock host(2):818: connection refused",
			want:   false,
		},
		"unrelated enoent in journal dump": {
			output: "listening on VSOCK port 818\nopen /etc/foo: no such file or directory",
			want:   false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isVsockUnavailableOutput(tc.output))
		})
	}
}

const scpBannerTimeoutStderr = "Connection timed out during banner exchange\nConnection to UNKNOWN port 65535 timed out\nscp: Connection closed\nexit status 255"

func TestWrapSCPError(t *testing.T) {
	t.Parallel()
	copyErr := errors.New("exit status 1")
	tests := map[string]struct {
		stderr        string
		wantTransport bool
		wantStalled   bool
		wantAuth      bool
	}{
		"banner timeout is SSH transport": {
			stderr:        scpBannerTimeoutStderr,
			wantTransport: true,
			wantStalled:   true,
		},
		"auth failure is terminal not transport": {
			stderr:   "Permission denied (publickey).\r\n",
			wantAuth: true,
		},
		"remote command failure is not SSH transport": {
			stderr: "install.sh: no such file\nexit status 1",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := wrapSCPError("/tmp/install.sh", "/guest/install.sh", tc.stderr, copyErr)
			require.Error(t, err)
			require.Equal(t, tc.wantTransport, errors.Is(err, errSSHTransport))
			require.Equal(t, tc.wantStalled, errors.Is(err, ErrSSHConnectivityStalled))
			require.Equal(t, tc.wantAuth, errors.Is(err, ErrSSHAuthenticationFailed))
		})
	}
}

func TestRetryCopyToGuest(t *testing.T) {
	t.Parallel()
	copyErr := errors.New("exit status 1")
	tests := map[string]struct {
		stderr           string
		succeedOnAttempt int
		wantAttempts     int
		wantErr          bool
		wantTransport    bool
		wantStalled      bool
		wantAuth         bool
	}{
		"retries banner timeout then succeeds": {
			stderr:           scpBannerTimeoutStderr,
			succeedOnAttempt: 3,
			wantAttempts:     3,
		},
		"gives up on persistent banner timeout": {
			stderr:        scpBannerTimeoutStderr,
			wantAttempts:  3,
			wantErr:       true,
			wantTransport: true,
			wantStalled:   true,
		},
		"does not retry terminal SSH auth": {
			stderr:       "Permission denied (publickey).\r\n",
			wantAttempts: 1,
			wantErr:      true,
			wantAuth:     true,
		},
		"does not retry remote command failure": {
			stderr:       "install.sh: permission denied",
			wantAttempts: 1,
			wantErr:      true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			attempts := 0
			err := retryCopyToGuest(t.Context(), nil, "/tmp/install.sh", "/guest/install.sh", 3, time.Millisecond, func() (string, error) {
				attempts++
				if tc.succeedOnAttempt > 0 && attempts >= tc.succeedOnAttempt {
					return "", nil
				}
				return tc.stderr, copyErr
			})
			require.Equal(t, tc.wantAttempts, attempts)
			if !tc.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, tc.wantTransport, errors.Is(err, errSSHTransport))
			require.Equal(t, tc.wantStalled, errors.Is(err, ErrSSHConnectivityStalled))
			require.Equal(t, tc.wantAuth, errors.Is(err, ErrSSHAuthenticationFailed))
		})
	}
}
