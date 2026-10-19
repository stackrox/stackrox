package vmagent_test

import (
	"context"
	"testing"

	"github.com/stackrox/rox/sensor/kubernetes/fake/vmagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetReport(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		numPackages    int
		enabled        bool
		lastKnownToken string
		wantUnchanged  bool
		wantPackages   int
		wantAgent      string
	}{
		"disabled returns unchanged": {
			numPackages:   10,
			enabled:       false,
			wantUnchanged: true,
		},
		"empty token returns full report": {
			numPackages:  7,
			enabled:      true,
			wantPackages: 7,
			wantAgent:    "fake",
		},
		"mismatched token returns full report": {
			numPackages:    7,
			enabled:        true,
			lastKnownToken: "stale",
			wantPackages:   7,
			wantAgent:      "fake",
		},
		"zero packages still returns a report": {
			numPackages:  0,
			enabled:      true,
			wantPackages: 0,
			wantAgent:    "fake",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := vmagent.NewClient(tt.numPackages, tt.enabled).GetReport(t.Context(), nil, tt.lastKnownToken)
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantUnchanged, result.Unchanged)

			if tt.wantUnchanged {
				assert.Nil(t, result.IndexReport)
				return
			}

			require.NotNil(t, result.IndexReport)
			assert.Equal(t, tt.wantPackages, len(result.IndexReport.GetContents().GetPackages()))
			require.NotNil(t, result.Meta)
			assert.NotEmpty(t, result.Meta.GetReportToken())
			assert.Equal(t, tt.wantAgent, result.Meta.GetAgentVersion())
		})
	}
}

func TestClient_GetReport_MatchingTokenReturnsUnchanged(t *testing.T) {
	t.Parallel()

	client := vmagent.NewClient(7, true)
	first, err := client.GetReport(t.Context(), nil, "")
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotEmpty(t, first.Meta.GetReportToken())

	second, err := client.GetReport(t.Context(), nil, first.Meta.GetReportToken())
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.True(t, second.Unchanged)
	assert.Nil(t, second.IndexReport)
}

func TestClient_GetReport_TokenStableAcrossFullReports(t *testing.T) {
	t.Parallel()

	client := vmagent.NewClient(7, true)
	a, err := client.GetReport(t.Context(), nil, "stale")
	require.NoError(t, err)
	require.NotNil(t, a)
	b, err := client.GetReport(t.Context(), nil, "stale")
	require.NoError(t, err)
	require.NotNil(t, b)
	assert.Equal(t, a.Meta.GetReportToken(), b.Meta.GetReportToken())
}

func TestClient_GetReport_CancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := vmagent.NewClient(7, true).GetReport(ctx, nil, "")
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDialThenGetReport(t *testing.T) {
	t.Parallel()

	stream, err := vmagent.NewDialer().Dial(t.Context(), "default", "vm-0", 1, true)
	require.NoError(t, err)
	require.NotNil(t, stream)
	t.Cleanup(func() {
		if stream != nil {
			_ = stream.Close()
		}
	})

	result, err := vmagent.NewClient(3, true).GetReport(t.Context(), stream, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.IndexReport)
	assert.Equal(t, 3, len(result.IndexReport.GetContents().GetPackages()))
}
