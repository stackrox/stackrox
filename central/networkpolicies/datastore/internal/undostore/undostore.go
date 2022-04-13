package undostore

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/bolthelper"
	"github.com/stackrox/stackrox/pkg/logging"
	bolt "go.etcd.io/bbolt"
)

var undoBucket = []byte("networkpolicies-undo")

var (
	log = logging.LoggerForModule()
)

// UndoStore provides storage functionality for undo records.
//go:generate mockgen-wrapper
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
