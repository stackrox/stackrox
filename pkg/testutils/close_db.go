package testutils

import (
	"os"

	"github.com/dgraph-io/badger"
	"go.etcd.io/bbolt"
)

// TearDownDB tears down an instance of BoltDB used in tests.
func TearDownDB(db *bbolt.DB) {
	_ = db.Close()
	_ = os.Remove(db.Path())
}

// TearDownBadger tears down an instance of BadgerDB used in tests.
func TearDownBadger(db *badger.DB, path string) {
	_ = db.Close()
	_ = os.Remove(path)
}
