package testutils

import (
	"os"

	"github.com/etcd-io/bbolt"
)

// TearDownDB tears down an instance of BoltDB used in tests.
func TearDownDB(db *bbolt.DB) {
	_ = db.Close()
	_ = os.Remove(db.Path())
}
