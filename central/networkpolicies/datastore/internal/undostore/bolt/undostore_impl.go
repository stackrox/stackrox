package bolt

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/metrics"
	"github.com/stackrox/stackrox/generated/storage"
	ops "github.com/stackrox/stackrox/pkg/metrics"
	bolt "go.etcd.io/bbolt"
)

type undoStore struct {
	db *bolt.DB
}

// GetNetworkPolicy returns network policy with given id.
func (s *undoStore) Get(_ context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "NetworkPolicyApplicationUndoRecord")
	clusterKey := []byte(clusterID)
	exists := false
	var record storage.NetworkPolicyApplicationUndoRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		if bucket == nil {
			// This should exist since we create it upon startup.
			return errors.New("top-level undo bucket not found")
		}
		val := bucket.Get(clusterKey)
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, &record)
	})
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	return &record, true, nil
}

func (s *undoStore) Upsert(ctx context.Context, record *storage.NetworkPolicyApplicationUndoRecord) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "NetworkPolicyApplicationUndoRecord")

	serialized, err := proto.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "serializing record")
	}

	clusterKey := []byte(record.GetClusterId())
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		if bucket == nil {
			// This should exist since we create it upon startup.
			return errors.New("top-level undo bucket not found")
		}
		return bucket.Put(clusterKey, serialized)
	})
}
