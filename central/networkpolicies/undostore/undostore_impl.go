package undostore

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoconv"
)

type undoStore struct {
	db *bolt.DB
}

func (s *undoStore) upsertNetworkPolicy(np *storage.NetworkPolicy) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		bytes, err := proto.Marshal(np)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(np.GetId()), bytes)
	})
}

// GetNetworkPolicy returns network policy with given id.
func (s *undoStore) GetUndoRecord(clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error) {
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

func (s *undoStore) UpsertUndoRecord(clusterID string, record *storage.NetworkPolicyApplicationUndoRecord) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "NetworkPolicyApplicationUndoRecord")

	serialized, err := proto.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "serializing record")
	}

	clusterKey := []byte(clusterID)
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		if bucket == nil {
			// This should exist since we create it upon startup.
			return errors.New("top-level undo bucket not found")
		}
		prevVal := bucket.Get(clusterKey)
		if prevVal != nil {
			var prevRecord storage.NetworkPolicyApplicationUndoRecord
			if err := proto.Unmarshal(prevVal, &prevRecord); err == nil {
				if protoconv.CompareProtoTimestamps(record.GetApplyTimestamp(), prevRecord.GetApplyTimestamp()) < 0 {
					return fmt.Errorf("apply timestamp of record to store (%v) is older than that of existing record (%v)",
						protoconv.ConvertTimestampToTimeOrDefault(record.GetApplyTimestamp(), time.Time{}),
						protoconv.ConvertTimestampToTimeOrDefault(prevRecord.GetApplyTimestamp(), time.Time{}))
				}
			}
		}

		return bucket.Put(clusterKey, serialized)
	})
}
