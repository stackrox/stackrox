package store

import (
	"time"

	bbolt "github.com/etcd-io/bbolt"
	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	storage "github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/storecache"
)

var (
	bucketName = []byte("processWhitelists2")
)

type store struct {
	crud protoCrud.MessageCrud
	db   *bbolt.DB
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.ProcessWhitelist).GetId())
}

func alloc() proto.Message {
	return new(storage.ProcessWhitelist)
}

func newStore(db *bbolt.DB, cache storecache.Cache) (*store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, bucketName, key, alloc)
	if err != nil {
		return nil, err
	}
	newCrud = protoCrud.NewCachedMessageCrud(newCrud, cache, "Whitelist", metrics.IncrementDBCacheCounter)
	return &store{crud: newCrud, db: db}, nil
}

func (s *store) AddWhitelist(whitelist *storage.ProcessWhitelist) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Whitelist")
	return s.crud.Create(whitelist)
}

func (s *store) DeleteWhitelist(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Whitelist")
	_, _, err := s.crud.Delete(id)
	return err
}

func (s *store) GetWhitelist(id string) (*storage.ProcessWhitelist, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Whitelist")
	msg, err := s.crud.Read(id)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	whitelist := msg.(*storage.ProcessWhitelist)
	return whitelist, nil
}

func (s *store) GetWhitelists(ids []string) ([]*storage.ProcessWhitelist, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Whitelist")
	msgs, missingIndices, err := s.crud.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	storedKeys := make([]*storage.ProcessWhitelist, 0, len(msgs))
	for _, msg := range msgs {
		storedKeys = append(storedKeys, msg.(*storage.ProcessWhitelist))
	}
	return storedKeys, missingIndices, nil
}

func (s *store) ListWhitelists() ([]*storage.ProcessWhitelist, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Whitelist")
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	storedKeys := make([]*storage.ProcessWhitelist, len(msgs))
	for i, msg := range msgs {
		storedKeys[i] = msg.(*storage.ProcessWhitelist)
	}
	return storedKeys, nil
}

func (s *store) UpdateWhitelist(whitelist *storage.ProcessWhitelist) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Whitelist")
	_, _, err := s.crud.Update(whitelist)
	return err
}

func (s *store) WalkAll(fn func(whitelist *storage.ProcessWhitelist) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Whitelist")
	return s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.ForEach(func(k, v []byte) error {
			var whitelist storage.ProcessWhitelist
			if err := proto.Unmarshal(v, &whitelist); err != nil {
				return err
			}

			if err := fn(&whitelist); err != nil {
				return err
			}

			return nil
		})
	})
}
