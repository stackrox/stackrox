package rocksdbmigration

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

func TestMigrateBadger(t *testing.T) {
	db := testutils.BadgerDBForT(t)
	defer func() { _ = db.Close() }()

	wb := db.NewWriteBatch()
	defer wb.Cancel()
	for i := 0; i < batchSize*10; i++ {
		num := fmt.Sprintf("%d", i)
		key := []byte(num)
		value := []byte(num)
		if err := wb.Set(key, value); err != nil {
			require.NoError(t, err)
		}
	}
	require.NoError(t, wb.Flush())

	rocksDB := rocksdbtest.RocksDBForT(t)
	defer rocksdbtest.TearDownRocksDB(rocksDB)

	err := migrateBadger(&types.Databases{
		BadgerDB: db,
		RocksDB:  rocksDB.DB,
	})
	require.NoError(t, err)

	tables := db.Tables(true)
	assert.Len(t, tables, 0)

	readOpts := gorocksdb.NewDefaultReadOptions()
	for i := 0; i < batchSize*10; i++ {
		num := fmt.Sprintf("%d", i)
		key := []byte(num)

		slice, err := rocksDB.Get(readOpts, key)
		require.NoError(t, err)
		require.True(t, slice.Exists())
		assert.Equal(t, key, slice.Copy())
	}
}
