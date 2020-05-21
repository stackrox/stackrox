package bolt

import (
	"time"

	bbolt "github.com/etcd-io/bbolt"
	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/rbac/k8srole/internal/store"
	storage "github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	ops "github.com/stackrox/rox/pkg/metrics"
	storecache "github.com/stackrox/rox/pkg/storecache"
)

var (
	bucketName = []byte("k8sroles")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.K8SRole).GetId())
}

func alloc() proto.Message {
	return new(storage.K8SRole)
}

// NewBoltStore returns a k8s role store backed by Bolt
func NewBoltStore(db *bbolt.DB, cache storecache.Cache) (store.Store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, bucketName, key, alloc)
	if err != nil {
		return nil, err
	}
	newCrud = protoCrud.NewCachedMessageCrud(newCrud, cache, "Role", metrics.IncrementDBCacheCounter)
	return &storeImpl{crud: newCrud}, nil
}

func (s *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Role")
	_, _, err := s.crud.Delete(id)
	return err
}

func (s *storeImpl) Get(id string) (*storage.K8SRole, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Role")
	msg, err := s.crud.Read(id)
	if err != nil {
		return nil, msg == nil, err
	}
	if msg == nil {
		return nil, false, nil
	}
	role := msg.(*storage.K8SRole)
	return role, true, nil
}

func (s *storeImpl) GetMany(ids []string) ([]*storage.K8SRole, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Role")
	msgs, missingIndices, err := s.crud.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	storedKeys := make([]*storage.K8SRole, 0, len(msgs))
	for _, msg := range msgs {
		storedKeys = append(storedKeys, msg.(*storage.K8SRole))
	}
	return storedKeys, missingIndices, nil
}

func (s *storeImpl) Walk(fn func(role *storage.K8SRole) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Role")
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err := fn(msg.(*storage.K8SRole)); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeImpl) Upsert(role *storage.K8SRole) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Role")
	_, _, err := s.crud.Upsert(role)
	return err
}
