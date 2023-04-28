// This file was originally generated with
// //go:generate cp ../../../../central/networkpolicies/datastore/internal/undostore/bolt/undostore_impl.go .

package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	bolt "go.etcd.io/bbolt"
)

type undoStore struct {
	db *bolt.DB
}

// Get returns network policy with given id.
func (s *undoStore) Get(_ context.Context, clusterID string) (*storage.NetworkPolicyApplicationUndoRecord, bool, error) {
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

func (s *undoStore) Upsert(_ context.Context, record *storage.NetworkPolicyApplicationUndoRecord) error {
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

func (s *undoStore) Walk(_ context.Context, fn func(np *storage.NetworkPolicyApplicationUndoRecord) error) error {
	return s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var np storage.NetworkPolicyApplicationUndoRecord
			if err := proto.Unmarshal(v, &np); err != nil {
				return err
			}
			return fn(&np)
		})
	})
}
