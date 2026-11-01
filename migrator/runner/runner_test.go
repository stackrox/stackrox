package runner

import (
	"testing"
	"time"

	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationCheckpointPreservesMetadata(t *testing.T) {
	source := migrations.MigrationVersion{MainVersion: "4.10.0", SeqNum: 220, MinimumSeqNum: 221, LastPersisted: time.Unix(123, 0)}
	checkpoint := migrationCheckpoint(source, 222, 220)
	assert.EqualValues(t, 222, checkpoint.GetSeqNum())
	assert.Equal(t, source.MainVersion, checkpoint.GetVersion())
	assert.EqualValues(t, 221, checkpoint.GetMinSeqNum())
	assert.Equal(t, source.LastPersisted.Unix(), checkpoint.GetLastPersisted().GetSeconds())
	assert.Equal(t, 220, source.SeqNum)
	source.MinimumSeqNum = 209
	assert.EqualValues(t, 220, migrationCheckpoint(source, 222, 220).GetMinSeqNum())
}

func TestMissingMigration(t *testing.T) {
	setMainVersion(t, "5.1.0")
	lookup := func(int) (types.Migration, bool) { return types.Migration{}, false }
	require.ErrorContains(t, runMigrations(&types.Databases{}, migrations.MigrationVersion{SeqNum: 220}, lookup), "no migration found starting at 220")
}

func TestInvalidTargetStopsMigrationWrites(t *testing.T) {
	for name, target := range map[string]string{
		"invalid version":  "garbage",
		"missing metadata": "5.5.0",
	} {
		t.Run(name, func(t *testing.T) {
			setMainVersion(t, target)
			_, expectedErr := migrations.MinimumSupportedForVersion(target)
			require.Error(t, expectedErr)
			require.EqualError(t, runMigrations(nil, migrations.MigrationVersion{SeqNum: 220}, nil), "determining minimum supported database sequence: "+expectedErr.Error())
			require.EqualError(t, UpdateToCurrentVersion(nil), "determining minimum supported database sequence: "+expectedErr.Error())
		})
	}
}

func setMainVersion(t *testing.T, target string) {
	t.Helper()
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	versiontest.SetMainVersion(t, target)
}
