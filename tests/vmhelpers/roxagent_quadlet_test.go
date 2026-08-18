//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"os"
	"path/filepath"
	"testing"

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

	got, err := overlayQuadletContainer(src, image, url)
	require.NoError(t, err)
	out := string(got)
	require.Contains(t, out, "Image="+image)
	require.NotContains(t, out, "Image=quay.io/stackrox-io/main:latest")
	require.Contains(t, out, "Exec=serve --host-path /host --rescan-interval "+E2ERescanInterval.String()+" --repo-cpe-url "+url)
	require.Contains(t, out, "Notify=true")
	require.Contains(t, out, "TimeoutStartSec=600")
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
			_, err := overlayQuadletContainer(tc.src, tc.image, tc.repo2cpeURL)
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
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, isVsockUnavailableOutput(tc.output))
		})
	}
}
