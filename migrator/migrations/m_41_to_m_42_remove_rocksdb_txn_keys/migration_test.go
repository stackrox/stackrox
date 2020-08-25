package m41tom42

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tecbot/gorocksdb"
)

func TestRemovePrefix(t *testing.T) {
	rocksDB, err := rocksdb.NewTemp(t.Name())
	require.NoError(t, err)
	defer rocksdbtest.TearDownRocksDB(rocksDB)

	wb := gorocksdb.NewWriteBatch()
	for i := 0; i < 5500; i++ {
		key := fmt.Sprintf("transactionsservice_accounts\x00%d", i)
		value := []byte("1")
		wb.Put([]byte(key), value)
	}
	require.NoError(t, rocksDB.Write(gorocksdb.NewDefaultWriteOptions(), wb))
	assert.NoError(t, removePrefix(rocksDB.DB, "service_accounts"))

	it := rocksDB.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	for it.Prev(); it.Valid(); it.Next() {
		assert.Fail(t, "RocksDB shouldn't contain any keys after remove prefix")
	}
}
