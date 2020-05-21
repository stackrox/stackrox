package bolt

import (
	"time"

	bbolt "github.com/etcd-io/bbolt"
	proto "github.com/gogo/protobuf/proto"
	metrics "github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/serviceaccount/internal/store"
	storage "github.com/stackrox/rox/generated/storage"
	protoCrud "github.com/stackrox/rox/pkg/bolthelper/crud/proto"
	ops "github.com/stackrox/rox/pkg/metrics"
	storecache "github.com/stackrox/rox/pkg/storecache"
)

var (
	bucketName = []byte("service_accounts")
)

type storeImpl struct {
	crud protoCrud.MessageCrud
}

func key(msg proto.Message) []byte {
	return []byte(msg.(*storage.ServiceAccount).GetId())
}

func alloc() proto.Message {
	return new(storage.ServiceAccount)
}

// NewBoltStore returns a new service account that uses BoltDB
func NewBoltStore(db *bbolt.DB, cache storecache.Cache) (store.Store, error) {
	newCrud, err := protoCrud.NewMessageCrud(db, bucketName, key, alloc)
	if err != nil {
		return nil, err
	}
	newCrud = protoCrud.NewCachedMessageCrud(newCrud, cache, "ServiceAccount", metrics.IncrementDBCacheCounter)
	return &storeImpl{crud: newCrud}, nil
}

func (s *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ServiceAccount")
	_, _, err := s.crud.Delete(id)
	return err
}

func (s *storeImpl) Get(id string) (*storage.ServiceAccount, bool, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ServiceAccount")
	msg, err := s.crud.Read(id)
	if err != nil {
		return nil, msg == nil, err
	}
	if msg == nil {
		return nil, false, nil
	}
	serviceaccount := msg.(*storage.ServiceAccount)
	return serviceaccount, true, nil
}

func (s *storeImpl) GetMany(ids []string) ([]*storage.ServiceAccount, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ServiceAccount")
	msgs, missingIndices, err := s.crud.ReadBatch(ids)
	if err != nil {
		return nil, nil, err
	}
	storedKeys := make([]*storage.ServiceAccount, 0, len(msgs))
	for _, msg := range msgs {
		storedKeys = append(storedKeys, msg.(*storage.ServiceAccount))
	}
	return storedKeys, missingIndices, nil
}

func (s *storeImpl) Walk(fn func(sa *storage.ServiceAccount) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "ServiceAccount")
	msgs, err := s.crud.ReadAll()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		if err := fn(msg.(*storage.ServiceAccount)); err != nil {
			return err
		}
	}
	return nil
}

func (s *storeImpl) Upsert(serviceaccount *storage.ServiceAccount) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "ServiceAccount")
	_, _, err := s.crud.Upsert(serviceaccount)
	return err
}
