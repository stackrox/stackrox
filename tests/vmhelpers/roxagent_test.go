//go:build test

package vmhelpers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVerboseOutputHasOSFields(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		stdout string
		want   bool
	}{
		"detectedOs key":       {`{"detectedOs":"rhel:9"}`, true},
		"operatingSystem key":  {`{"operatingSystem":"rhel"}`, true},
		"operating_system key": {`{"operating_system":"rhel"}`, true},
		"no OS keys":           {`{"components":["rpm"]}`, false},
		"empty":                {``, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, verboseOutputHasOSFields(tc.stdout))
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

func TestIsTerminalVSOCKUnavailableError(t *testing.T) {
	t.Parallel()

	terminalErr := errors.Join(ErrTerminalVSOCKUnavailable, errors.New("exit status 1"))
	require.True(t, IsTerminalVSOCKUnavailableError(terminalErr))
	require.False(t, IsTerminalVSOCKUnavailableError(errors.New("other error")))
}
