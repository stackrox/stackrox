package migrations

import (
	"testing"

	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeqNumForVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		target   productstreams.XYVersion
		sequence int
		wantErr  string
	}{
		"current stream":        {target: productstreams.XYVersion{X: 5, Y: 1}, sequence: 227},
		"major boundary":        {target: productstreams.XYVersion{X: 5, Y: 0}, sequence: 227},
		"previous Y-stream":     {target: productstreams.XYVersion{X: 4, Y: 11}, sequence: 225},
		"N-2":                   {target: productstreams.XYVersion{X: 4, Y: 10}, sequence: 220},
		"shared sequence":       {target: productstreams.XYVersion{X: 4, Y: 6}, sequence: 209},
		"unknown older version": {target: productstreams.XYVersion{X: 4, Y: 3}, wantErr: "Failed to find sequence number"},
		"unknown newer version": {target: productstreams.XYVersion{X: 99, Y: 99}, wantErr: "Failed to find sequence number"},
	} {
		t.Run(name, func(t *testing.T) {
			sequence, err := SeqNumForVersion(tc.target)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.Zero(t, sequence)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.sequence, sequence)
		})
	}
}

func TestSeqNumForVersionMatchesReleaseMetadata(t *testing.T) {
	for _, release := range releaseVersions {
		target, err := productstreams.ParseXYFromVersionString(release.Version)
		require.NoError(t, err)
		sequence, err := SeqNumForVersion(target)
		require.NoError(t, err)
		assert.Equal(t, release.Sequence, sequence, release.Version)
	}
}
