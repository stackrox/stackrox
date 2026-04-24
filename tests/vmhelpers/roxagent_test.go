//go:build test

package vmhelpers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChooseRepo2CPESource(t *testing.T) {
	t.Parallel()
	primary := "https://remote/repo2cpe.json"
	fallback := "file:///var/lib/rox/repo2cpe.json"
	cases := map[string]struct {
		attemptsMade, maxPrimary int
		want                     string
	}{
		"zero attempts uses primary when budget positive":  {0, 5, primary},
		"mid range still primary":                          {2, 5, primary},
		"last primary slot before boundary":                {4, 5, primary},
		"boundary falls back when attemptsMade equals max": {5, 5, fallback},
		"plan boundary three of three":                     {3, 3, fallback},
		"zero max immediate fallback on first index":       {0, 0, fallback},
		"after boundary stays fallback":                    {10, 3, fallback},
		"single primary slot then fallback":                {1, 1, fallback},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := chooseRepo2CPESource(tc.attemptsMade, tc.maxPrimary, primary, fallback)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestRunRoxagentOnce_RejectsUnsupportedInstallPath(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, err := RunRoxagentOnce(ctx, Virtctl{}, "ns", "vm", RoxagentRunConfig{
		RoxagentInstallPath: "/opt/roxagent",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not supported")
	require.Contains(t, err.Error(), "/opt/roxagent")
}

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

func TestVerboseOutputLooksLikeReport_FalseForQuietOutput(t *testing.T) {
	t.Parallel()
	require.False(t, VerboseOutputLooksLikeReport("done"))
}

func TestVerboseOutputLooksLikeReport_TrueForSampleReportJSON(t *testing.T) {
	t.Parallel()
	sample := `{"scan":{"operatingSystem":"rhel"},"components":[{"name":"a"}]}`
	require.True(t, VerboseOutputLooksLikeReport(sample))
}

func TestVerboseOutputLooksLikeReport_TrueForIndexReportJSON(t *testing.T) {
	t.Parallel()
	sample := `{"indexReport":{"state":"IndexFinished"},"discoveredData":{"operatingSystem":{"name":"rhel"}}}`
	require.True(t, VerboseOutputLooksLikeReport(sample))
}

func TestBuildRoxagentInstallArgs_UsesSudoInstallModeAndPaths(t *testing.T) {
	t.Parallel()
	require.Equal(t,
		[]string{"sudo", "install", "-m", "0755", "/tmp/roxagent", "/usr/local/bin/roxagent"},
		buildRoxagentInstallArgs("/tmp/roxagent", "/usr/local/bin/roxagent"),
	)
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
