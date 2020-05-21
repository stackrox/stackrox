package m36tom37

import (
	"fmt"
	"os"
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

func TestMigrateBolt(t *testing.T) {
	db := testutils.DBForT(t)
	defer func() { _ = db.Close() }()

	err := db.Update(func(tx *bbolt.Tx) error {
		bucket, err := tx.CreateBucket([]byte("risk"))
		if err != nil {
			return err
		}
		for i := 0; i < batchSize*10; i++ {
			num := fmt.Sprintf("%d", i)
			key := []byte(num)
			value := []byte(num)
			if err := bucket.Put(key, value); err != nil {
				return err
			}
		}
		return nil
	})
	require.NoError(t, err)

	rocksDB, dir, err := rocksdb.NewTemp(t.Name())
	require.NoError(t, err)
	func() { _ = os.RemoveAll(dir) }()

	err = migrateBolt(&types.Databases{
		BoltDB:  db,
		RocksDB: rocksDB,
	})
	require.NoError(t, err)

	readOpts := gorocksdb.NewDefaultReadOptions()
	for i := 0; i < batchSize*10; i++ {
		key := []byte(fmt.Sprintf("risk\x00%d", i))
		value := []byte(fmt.Sprintf("%d", i))

		slice, err := rocksDB.Get(readOpts, key)
		require.NoError(t, err)
		require.True(t, slice.Exists())
		assert.Equal(t, value, slice.Copy())
	}
}
