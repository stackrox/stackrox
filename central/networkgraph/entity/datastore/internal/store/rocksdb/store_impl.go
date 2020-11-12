package rocksdb

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/db/mapcache"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
)

var (
	bucket = []byte("networkentity")
)

type storeImpl struct {
	crud db.Crud
}

func alloc() proto.Message {
	return &storage.NetworkEntity{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.NetworkEntity).GetInfo().GetId())
}

func uniqKeyFunc(msg proto.Message) []byte {
	entity := msg.(*storage.NetworkEntity)
	uniqKey := entity.GetScope().GetClusterId() + ":" + entity.GetInfo().GetExternalSource().GetCidr()
	return []byte(uniqKey)
}

// New returns a new Store instance using the provided rocksdb instance.
func New(db *rocksdb.RocksDB) (store.EntityStore, error) {
	globaldb.RegisterBucket(bucket, "NetworkEntity")
	uniqKeyCrud := generic.NewUniqueKeyCRUD(db, bucket, keyFunc, alloc, uniqKeyFunc, false)
	cacheCrud, err := mapcache.NewMapCache(uniqKeyCrud, keyFunc)
	if err != nil {
		return nil, err
	}

	return &storeImpl{
		crud: cacheCrud,
	}, nil
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs() ([]string, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetAll, "NetworkEntityIDs")

	return s.crud.GetKeys()
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(id string) (bool, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Exists, "NetworkEntity")

	return s.crud.Exists(id)
}

func (s *storeImpl) Get(id string) (*storage.NetworkEntity, bool, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Get, "NetworkEntity")

	msg, exists, err := s.crud.Get(id)
	if err != nil || !exists {
		return nil, false, err
	}
	return msg.(*storage.NetworkEntity), true, nil
}

func (s *storeImpl) Upsert(entity *storage.NetworkEntity) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Add, "NetworkEntity")

	return s.crud.Upsert(entity)
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(id string) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Remove, "NetworkEntity")

	return s.crud.Delete(id)
}

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ids []string) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.RemoveMany, "NetworkEntity")

	return s.crud.DeleteMany(ids)
}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(fn func(obj *storage.NetworkEntity) error) error {
	return s.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*storage.NetworkEntity))
	})
}
