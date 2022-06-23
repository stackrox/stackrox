package bolt

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	bolt "go.etcd.io/bbolt"
)

func alloc() proto.Message {
	return new(storage.ExternalBackup)
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.ExternalBackup).GetId())
}

var (
	backupBucketKey = []byte("externalBackups")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

// New returns a new Node store
func New(db *bolt.DB) *storeImpl {
	bolthelper.RegisterBucketOrPanic(db, backupBucketKey)

	crud := protoCrud.NewMessageCrudForBucket(bolthelper.TopLevelRef(db, backupBucketKey), key, alloc)
	return &storeImpl{crud: crud}
}

func (s *storeImpl) GetAll(_ context.Context) ([]*storage.ExternalBackup, error) {
	entries, err := s.crud.ReadAll()
	if err != nil {
		return nil, err
	}
	backups := make([]*storage.ExternalBackup, len(entries))
	for i, entry := range entries {
		backups[i] = entry.(*storage.ExternalBackup)
	}
	return backups, nil
}

func (s *storeImpl) Get(_ context.Context, id string) (*storage.ExternalBackup, bool, error) {
	value, err := s.crud.Read(id)
	if err != nil || value == nil {
		return nil, false, err
	}
	return value.(*storage.ExternalBackup), true, nil
}

func (s *storeImpl) Upsert(_ context.Context, backup *storage.ExternalBackup) error {
	_, _, err := s.crud.Upsert(backup)
	return err
}

func (s *storeImpl) Delete(_ context.Context, id string) error {
	_, _, err := s.crud.Delete(id)
	return err
}
