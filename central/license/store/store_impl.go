package store

import (
	bolt "github.com/etcd-io/bbolt"
	proto2 "github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	bucketID = []byte("licenseKeys")
)

type store struct {
	crud proto.MessageCrud
}

func key(msg proto2.Message) []byte {
	return []byte(msg.(*storage.StoredLicenseKey).GetLicenseId())
}

func alloc() proto2.Message {
	return new(storage.StoredLicenseKey)
}

func newStore(db *bolt.DB) (*store, error) {
	if err := bolthelper.RegisterBucket(db, bucketID); err != nil {
		return nil, err
	}

	return &store{
		crud: proto.NewMessageCrud(db, bucketID, key, alloc),
	}, nil
}

func (s *store) ListLicenseKeys() ([]*storage.StoredLicenseKey, error) {
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}

	storedKeys := make([]*storage.StoredLicenseKey, len(msgs))
	for i, msg := range msgs {
		storedKeys[i] = msg.(*storage.StoredLicenseKey)
	}

	return storedKeys, nil
}

func (s *store) UpsertLicenseKeys(keys []*storage.StoredLicenseKey) error {
	msgs := make([]proto2.Message, len(keys))
	for i, key := range keys {
		msgs[i] = key
	}
	return s.crud.UpsertBatch(msgs)
}

func (s *store) DeleteLicenseKey(licenseID string) error {
	return s.crud.Delete(licenseID)
}
