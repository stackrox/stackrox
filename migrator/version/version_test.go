//go:build sql_integration

package version

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestReadVersionFreshDatabase(t *testing.T) {
	db := openVersionDB(t)
	ver := readVersion(t, db)
	assert.Equal(t, "0", ver.MainVersion)
	assert.Zero(t, ver.SeqNum)
	assert.False(t, db.Migrator().HasTable("versions"), "reading must not mutate the schema")
}

func TestReadVersionColumnMetadata(t *testing.T) {
	db := openVersionDB(t)
	require.NoError(t, db.Exec("CREATE TABLE versions (seqnum integer, version varchar, minseqnum integer, lastpersisted timestamp)").Error)
	persisted := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	require.NoError(t, db.Exec("INSERT INTO versions VALUES (?, ?, ?, ?)", 220, "4.10.0", 209, persisted).Error)
	ver := readVersion(t, db)
	assert.Equal(t, "4.10.0", ver.MainVersion)
	assert.Equal(t, 220, ver.SeqNum)
	assert.Equal(t, 209, ver.MinimumSeqNum)
	assert.True(t, persisted.Equal(ver.LastPersisted))
	assert.False(t, db.Migrator().HasColumn("versions", "serialized"))
}

func TestReadVersionMissingMetadataIsFresh(t *testing.T) {
	for name, tableExists := range map[string]bool{"missing table": false, "empty table": true} {
		t.Run(name, func(t *testing.T) {
			db := openVersionDB(t)
			require.NoError(t, db.Exec("CREATE TABLE existing_data (id integer)").Error)
			if tableExists {
				require.NoError(t, db.Exec("CREATE TABLE versions (seqnum integer, version varchar, minseqnum integer, lastpersisted timestamp)").Error)
			}
			ver := readVersion(t, db)
			assert.Equal(t, "0", ver.MainVersion)
			assert.Zero(t, ver.SeqNum)
			assert.True(t, db.Migrator().HasTable("existing_data"))
			assert.Equal(t, tableExists, db.Migrator().HasTable("versions"), "reading must not change the schema")
		})
	}
}

func TestReadVersionCanceledContext(t *testing.T) {
	t.Setenv(env.UnsafeAllowUnsupportedUpgrade.EnvVar(), "true")
	db := openVersionDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ReadVersionGormDB(ctx, db)
	require.Error(t, err)
}

func TestReadVersionIgnoresObsoleteSerializedData(t *testing.T) {
	t.Setenv(env.UnsafeAllowUnsupportedUpgrade.EnvVar(), "true")
	db := openVersionDB(t)
	SetVersion(context.Background(), db, &storage.Version{SeqNum: 220, Version: "4.10.0"}, true)
	require.NoError(t, db.Exec("UPDATE versions SET serialized = ?", []byte{0xff}).Error)
	ver, err := ReadVersionGormDB(context.Background(), db)
	require.NoError(t, err, "obsolete serialized data must not affect modern metadata")
	assert.Equal(t, "4.10.0", ver.MainVersion)
	assert.Equal(t, 220, ver.SeqNum)
}

func TestReadVersionMissingColumn(t *testing.T) {
	t.Setenv(env.UnsafeAllowUnsupportedUpgrade.EnvVar(), "true")
	db := openVersionDB(t)
	SetVersion(context.Background(), db, &storage.Version{SeqNum: 220, Version: "4.10.0"}, true)
	require.NoError(t, db.Exec("ALTER TABLE versions DROP COLUMN seqnum").Error)
	_, err := ReadVersionGormDB(context.Background(), db)
	require.Error(t, err, "the override must not swallow query failures")
}

func TestReadVersionLegacyMetadataRejected(t *testing.T) {
	for name, override := range map[string]string{"normal": "false", "override": "true"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.UnsafeAllowUnsupportedUpgrade.EnvVar(), override)
			db := openVersionDB(t)
			require.NoError(t, db.Exec("CREATE TABLE versions (serialized bytea)").Error)
			_, err := ReadVersionGormDB(context.Background(), db)
			require.ErrorContains(t, err, "reading version metadata; manual recovery required")
		})
	}
}

func TestReadVersionInvalidRecordsAreNotFresh(t *testing.T) {
	t.Setenv(env.UnsafeAllowUnsupportedUpgrade.EnvVar(), "true")
	for name, records := range map[string][]*storage.Version{
		"zero record":      {{Version: "0"}},
		"negative minimum": {{Version: "4.10.0", SeqNum: 220, MinSeqNum: -1}},
		"multiple records": {{Version: "4.10.0", SeqNum: 220}, {Version: "4.11.0", SeqNum: 225}},
	} {
		t.Run(name, func(t *testing.T) {
			db := openVersionDB(t)
			SetVersion(context.Background(), db, records[0], true)
			for _, record := range records[1:] {
				require.NoError(t, db.Exec("INSERT INTO versions (seqnum, version) VALUES (?, ?)", record.GetSeqNum(), record.GetVersion()).Error)
			}
			_, err := ReadVersionGormDB(context.Background(), db)
			require.Error(t, err)
		})
	}
}

func TestReadVersionMissingMetadataWithExistingViewIsFresh(t *testing.T) {
	db := openVersionDB(t)
	require.NoError(t, db.Exec("CREATE VIEW existing_view AS SELECT 1 AS id").Error)
	ver := readVersion(t, db)
	assert.Equal(t, "0", ver.MainVersion)
	assert.Zero(t, ver.SeqNum)
	assert.False(t, db.Migrator().HasTable("versions"))
}

func openVersionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
	t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
	return db
}

func readVersion(t *testing.T, db *gorm.DB) *migrations.MigrationVersion {
	t.Helper()
	ver, err := ReadVersionGormDB(context.Background(), db)
	require.NoError(t, err)
	return ver
}
