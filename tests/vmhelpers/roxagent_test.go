//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
