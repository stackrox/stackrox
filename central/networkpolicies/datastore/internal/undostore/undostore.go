package undostore

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/logging"
)

var undoBucket = []byte("networkpolicies-undo")

var (
	log = logging.LoggerForModule()
)

// UndoStore provides storage functionality for undo records.
type UndoStore interface {
	GetUndoRecord(clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error)
	UpsertUndoRecord(clusterID string, undoRecord *storage.NetworkPolicyApplicationUndoRecord) error
}

// New returns a new UndoStore instance using the provided bolt DB instance.
func New(db *bolt.DB) UndoStore {
	bolthelper.RegisterBucketOrPanic(db, undoBucket)
	return &undoStore{
		db: db,
	}
}
