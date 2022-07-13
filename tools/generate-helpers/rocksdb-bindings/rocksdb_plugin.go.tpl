
package legacy
import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/db"
	{{- if .Cache}}
	"github.com/stackrox/rox/pkg/db/mapcache"
	{{- end}}
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
)

var (
	bucket = []byte("{{.Bucket}}")
)

type Store interface {
    UpsertMany(ctx context.Context, objs []*storage.{{.Type}}) error
	Walk(ctx context.Context, fn func(obj *storage.{{.Type}}) error) error
}

type storeImpl struct {
	crud db.Crud
}

func alloc() proto.Message {
	return &storage.{{.Type}}{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.{{.Type}}).{{.KeyFunc}})
}

{{- if .UniqKeyFunc}}
func uniqKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.{{.Type}}).{{.UniqKeyFunc}})
}
{{- end}}

// New returns a new Store instance using the provided rocksdb instance.
func New(db *rocksdb.RocksDB) (Store, error) {
	{{- if .UniqKeyFunc}}
	baseCRUD := generic.NewUniqueKeyCRUD(db, bucket, keyFunc, alloc, uniqKeyFunc, {{.TrackIndex}})
	{{- else}}
	baseCRUD := generic.NewCRUD(db, bucket, keyFunc, alloc, {{.TrackIndex}})
	{{- end}}
    {{- if not .Cache}}
    return  &storeImpl{crud: baseCRUD}, nil
    {{- else}}
	cacheCRUD, err := mapcache.NewMapCache(baseCRUD, keyFunc)
	if err != nil {
		return nil, err
	}
	return &storeImpl{
		crud: cacheCRUD,
	}, nil
    {{- end}}
}

// UpsertMany batches objects into the DB
func (b *storeImpl) UpsertMany(_ context.Context, objs []*storage.{{.Type}}) error {
	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
    }

	return b.crud.UpsertMany(msgs)
}

// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *storage.{{.Type}}) error) error {
	return b.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*storage.{{.Type}}))
	})
}
