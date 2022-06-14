package bolt

import (
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/logging"
	bolt "go.etcd.io/bbolt"
)

var undoBucket = []byte("networkpolicies-undo")

var (
	log = logging.LoggerForModule()
)

// New returns a new UndoStore instance using the provided bolt DB instance.
func New(db *bolt.DB) *undoStore {
	bolthelper.RegisterBucketOrPanic(db, undoBucket)
	return &undoStore{
		db: db,
	}
}
