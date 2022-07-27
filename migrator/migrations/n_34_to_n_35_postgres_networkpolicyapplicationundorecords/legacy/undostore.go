// This file was originally generated with
// //go:generate cp ../../../../central/networkpolicies/datastore/internal/undostore/bolt/undostore.go .

package legacy

import (
	"github.com/stackrox/rox/pkg/bolthelper"
	bolt "go.etcd.io/bbolt"
)

var undoBucket = []byte("networkpolicies-undo")

// New returns a new UndoStore instance using the provided bolt DB instance.
func New(db *bolt.DB) *undoStore {
	bolthelper.RegisterBucketOrPanic(db, undoBucket)
	return &undoStore{
		db: db,
	}
}
