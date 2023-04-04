package legacy

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
)

var (
	bucket = []byte("clusters_health_status")
)

// Store implements the methods used for migrating cluster health statuses
type Store interface {
	UpsertMany(ctx context.Context, objs []*storage.ClusterHealthStatus) error
	Walk(ctx context.Context, fn func(obj *storage.ClusterHealthStatus) error) error
}

type storeImpl struct {
	crud db.Crud
}

func alloc() proto.Message {
	return &storage.ClusterHealthStatus{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.ClusterHealthStatus).GetId())
}
func uniqKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.ClusterHealthStatus).GetId())
}

// New returns a new Store instance using the provided rocksdb instance.
func New(db *rocksdb.RocksDB) (Store, error) {
	baseCRUD := generic.NewUniqueKeyCRUD(db, bucket, keyFunc, alloc, uniqKeyFunc, false)
	return &storeImpl{crud: baseCRUD}, nil
}

// UpsertMany batches objects into the DB
func (b *storeImpl) UpsertMany(_ context.Context, objs []*storage.ClusterHealthStatus) error {
	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
	}

	return b.crud.UpsertMany(msgs)
}

// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *storage.ClusterHealthStatus) error) error {
	return b.crud.WalkAllWithID(func(id []byte, msg proto.Message) error {
		chs := msg.(*storage.ClusterHealthStatus)
		if chs.GetId() == "" {
			chs.Id = string(id)
		}
		return fn(chs)
	})
}
