//go:build sql_integration

package version

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadVersionFreshDatabase(t *testing.T) {
	db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
	t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
	ver, err := ReadVersionGormDB(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, "0", ver.MainVersion)
	assert.Zero(t, ver.SeqNum)
	assert.False(t, db.Migrator().HasTable("versions"), "reading must not mutate the schema")
}

func TestReadVersionMissingMetadataInPopulatedDatabase(t *testing.T) {
	db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
	t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
	require.NoError(t, db.Exec("CREATE TABLE existing_data (id integer)").Error)
	_, err := ReadVersionGormDB(context.Background(), db)
	require.ErrorContains(t, err, "missing version metadata")
}

func TestReadVersionPropagatesErrors(t *testing.T) {
	db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
	t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ReadVersionGormDB(ctx, db)
	require.Error(t, err)

	SetVersion(context.Background(), db, &storage.Version{SeqNum: 220, Version: "4.10.0"}, true)
	require.NoError(t, db.Exec("UPDATE versions SET serialized = ?", []byte{0xff}).Error)
	_, err = ReadVersionGormDB(context.Background(), db)
	require.ErrorContains(t, err, "decoding")
}

func TestReadVersionLegacySerializedMetadata(t *testing.T) {
	db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
	t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
	require.NoError(t, db.Exec("CREATE TABLE versions (serialized bytea)").Error)
	data, err := (&storage.Version{SeqNum: 220, Version: "4.10.0"}).MarshalVT()
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO versions VALUES (decode(?, 'hex'))", hex.EncodeToString(data)).Error)
	ver, err := ReadVersionGormDB(context.Background(), db)
	require.NoError(t, err)
	assert.Equal(t, "4.10.0", ver.MainVersion)
	assert.Equal(t, 220, ver.SeqNum)
}

func TestReadVersionInvalidRecordsAreNotFresh(t *testing.T) {
	for name, records := range map[string][]*storage.Version{
		"zero record":      {{Version: "0"}},
		"negative minimum": {{Version: "4.10.0", SeqNum: 220, MinSeqNum: -1}},
		"multiple records": {{Version: "4.10.0", SeqNum: 220}, {Version: "4.11.0", SeqNum: 225}},
	} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.OpenGormDB(t, pgtest.GetConnectionString(t))
			t.Cleanup(func() { pgtest.CloseGormDB(t, db) })
			SetVersion(context.Background(), db, records[0], true)
			for _, record := range records[1:] {
				require.NoError(t, db.Exec("INSERT INTO versions (seqnum, version) VALUES (?, ?)", record.GetSeqNum(), record.GetVersion()).Error)
			}
			_, err := ReadVersionGormDB(context.Background(), db)
			require.Error(t, err)
		})
	}
}
