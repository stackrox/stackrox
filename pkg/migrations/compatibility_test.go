package migrations

import (
	"testing"

	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinimumSupportedForVersion(t *testing.T) {
	for name, tc := range map[string]struct {
		target, minimum string
		sequence        int
	}{
		"major boundary": {"5.0.0", "4.9", 213},
		"current":        {"5.1.x-42-gabcdef", "4.10", 220},
		"patch":          {"5.1.3", "4.10", 220},
		"next":           {"5.2.0-rc.1", "4.11", 225},
	} {
		t.Run(name, func(t *testing.T) {
			minimum, err := MinimumSupportedForVersion(tc.target)
			require.NoError(t, err)
			assert.Equal(t, tc.minimum, minimum.Version)
			assert.Equal(t, tc.sequence, minimum.Sequence)
		})
	}
	for _, target := range []string{"", "garbage", "-1.2", "5.-1", "5.5.0"} {
		_, err := MinimumSupportedForVersion(target)
		require.Error(t, err, target)
	}
}

func TestMinimumGettersStayInSync(t *testing.T) {
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	versiontest.SetMainVersion(t, "5.1.0")
	minimum, err := MinimumSupportedDBVersion()
	require.NoError(t, err)
	assert.Equal(t, "4.10", minimum)
	sequence, err := MinimumSupportedDBVersionSeqNum()
	require.NoError(t, err)
	assert.Equal(t, 220, sequence)
}

func TestMinimumGettersInvalidTarget(t *testing.T) {
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	for name, target := range map[string]string{
		"missing version":  "",
		"invalid version":  "garbage",
		"missing metadata": "5.5.0",
	} {
		t.Run(name, func(t *testing.T) {
			versiontest.SetMainVersion(t, target)
			_, expectedErr := MinimumSupportedForVersion(target)
			require.Error(t, expectedErr)
			minimum, err := MinimumSupportedDBVersion()
			require.EqualError(t, err, expectedErr.Error())
			assert.Empty(t, minimum)
			sequence, err := MinimumSupportedDBVersionSeqNum()
			require.EqualError(t, err, expectedErr.Error())
			assert.Zero(t, sequence)
		})
	}
}

func TestCheckUpgradeCompatibility(t *testing.T) {
	for name, tc := range map[string]struct {
		source          string
		seq, minimum    int
		unsafe, blocked bool
	}{
		"N-3":                     {source: "4.10.0", seq: 220},
		"N-2":                     {source: "4.11.3", seq: 225},
		"N-1":                     {source: "5.0.x-1-gabc", seq: 227},
		"same stream":             {source: "5.1.2", seq: 227},
		"N-4 same sequence":       {source: "4.9.0", seq: 227, blocked: true},
		"missing source":          {seq: 220, blocked: true},
		"malformed source":        {source: "invalid", seq: 220, blocked: true},
		"override old source":     {source: "4.9.0", seq: 213, unsafe: true},
		"override missing source": {seq: 220, unsafe: true},
		"override cannot bypass rollback minimum": {source: "5.2.0", seq: 230, minimum: 228, unsafe: true, blocked: true},
		"supported rollback":                      {source: "5.2.0", seq: 230, minimum: 220},
		"fresh":                                   {source: "0"},
	} {
		t.Run(name, func(t *testing.T) {
			bypassed, err := CheckUpgradeCompatibility(MigrationVersion{MainVersion: tc.source, SeqNum: tc.seq, MinimumSeqNum: tc.minimum}, "5.1.0", 227, tc.unsafe)
			if tc.blocked {
				var compatibilityErr *CompatibilityError
				require.ErrorAs(t, err, &compatibilityErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.unsafe && !tc.blocked, bypassed != "")
		})
	}
}

func TestCompatibilityMessage(t *testing.T) {
	_, err := CheckUpgradeCompatibility(MigrationVersion{MainVersion: "4.4.0", SeqNum: 199}, "5.2.0", 227, false)
	require.ErrorContains(t, err, "4.4")
	require.ErrorContains(t, err, "5.2")
	require.ErrorContains(t, err, "4.11")
	require.ErrorContains(t, err, "4.7")
}

func TestReleaseSequenceMetadata(t *testing.T) {
	var previous string
	lastSequence := 0
	for _, release := range releaseVersions {
		stream, err := parseStream(release.Version)
		require.NoError(t, err)
		require.Equal(t, stream.String(), release.Version)
		require.Positive(t, release.Sequence)
		require.GreaterOrEqual(t, release.Sequence, lastSequence)
		if previous != "" {
			last, err := parseStream(previous)
			require.NoError(t, err)
			require.Positive(t, stream.Compare(last))
		}
		previous, lastSequence = release.Version, release.Sequence
	}
	releases := []ReleaseVersion{{Version: "4.6", Sequence: 209}, {Version: "4.7", Sequence: 209}, {Version: "5.2", Sequence: 300}}
	minimum, err := minimumSupportedForVersion("5.5.0", releases)
	require.NoError(t, err)
	assert.Equal(t, ReleaseVersion{Version: "5.2", Sequence: 300}, minimum)
}

func TestUnsafeOverrideCannotBypassInvalidTarget(t *testing.T) {
	for _, target := range []string{"", "5.5.0"} {
		_, err := CheckUpgradeCompatibility(MigrationVersion{MainVersion: "4.9.0", SeqNum: 213}, target, 227, true)
		require.Error(t, err)
	}
}
