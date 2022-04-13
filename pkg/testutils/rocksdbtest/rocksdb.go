package rocksdbtest

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// This package exists separately from testutils because then RocksDB is not imported for those packages
// and they can be built statically

// RocksDBForT creates and returns a RocksDB for the test
func RocksDBForT(t testing.TB) *rocksdb.RocksDB {
	db, err := rocksdb.NewTemp(testutils.DBFileNameForT(t))
	require.NoError(t, err)
	require.NotNil(t, db)
	return db
}

// TearDownRocksDB tears down a RocksDB instance used in tests
func TearDownRocksDB(db *rocksdb.RocksDB) {
	_ = rocksdb.CloseAndRemove(db)
}
