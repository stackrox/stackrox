package rocksdbtest

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// This package exists separately from testutils because then RocksDB is not imported for those packages
// and they can be built statically

// RocksDBForT creates and returns a RocksDB for the test
func RocksDBForT(t *testing.T) *rocksdb.RocksDB {
	db, _, err := rocksdb.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)
	require.NotNil(t, db)
	return db
}

// TearDownRocksDB tears down a RocksDB instance used in tests
func TearDownRocksDB(db *rocksdb.RocksDB, path string) {
	db.Close()
	_ = os.Remove(path)
}
