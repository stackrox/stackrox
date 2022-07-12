
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
	{{- if .NoKeyField}}
    UpsertManyWithIDs(ctx context.Context, ids []string, objs []*storage.{{.Type}}) error
    {{- else }}
    UpsertMany(ctx context.Context, objs []*storage.{{.Type}}) error
    {{- end}}
	{{- if .NoKeyField}}
	WalkAllWithID(ctx context.Context, fn func(id string, obj *storage.{{.Type}}) error) error
	{{- else }}
	Walk(ctx context.Context, fn func(obj *storage.{{.Type}}) error) error
	{{- end}}
}

type storeImpl struct {
	crud db.Crud
}

func alloc() proto.Message {
	return &storage.{{.Type}}{}
}
{{ if not .NoKeyField}}
func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.{{.Type}}).{{.KeyFunc}})
}
{{- end}}

{{- if .UniqKeyFunc}}
func uniqKeyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.{{.Type}}).{{.UniqKeyFunc}})
}
{{- end}}

// New returns a new Store instance using the provided rocksdb instance.
func New(db *rocksdb.RocksDB) (Store, error) {
//	globaldb.RegisterBucket(bucket, "{{.Type}}")
	{{- if .UniqKeyFunc}}
	baseCRUD := generic.NewUniqueKeyCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, alloc, uniqKeyFunc, {{.TrackIndex}})
	{{- else}}
	baseCRUD := generic.NewCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, alloc, {{.TrackIndex}})
	{{- end}}
    {{- if not .Cache}}
    return  &storeImpl{crud: baseCRUD}, nil
    {{- else}}
	cacheCRUD, err := mapcache.NewMapCache(baseCRUD, {{if .NoKeyField}}nil{{else}}keyFunc{{end}})
	if err != nil {
		return nil, err
	}
	return &storeImpl{
		crud: cacheCRUD,
	}, nil
    {{- end}}
}

{{- if .NoKeyField}}
// UpsertManyWithIDs batches objects into the DB
func (b *storeImpl) UpsertManyWithIDs(_ context.Context, ids []string, objs []*storage.{{.Type}}) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")

	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
    }

	return b.crud.UpsertManyWithIDs(ids, msgs)

{{- else}}
// UpsertMany batches objects into the DB
func (b *storeImpl) UpsertMany(_ context.Context, objs []*storage.{{.Type}}) error {
	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
    }

	return b.crud.UpsertMany(msgs)
}
{{- end}}

{{- if .NoKeyField}}
// WalkAllWithID iterates over all of the objects in the store and applies the closure
func (b *storeImpl) WalkAllWithID(_ context.Context, fn func(id string, obj *storage.{{.Type}}) error) error {
	return b.crud.WalkAllWithID(func(id []byte, msg proto.Message) error {
		return fn(string(id), msg.(*storage.{{.Type}}))
	})
}
{{- else}}
// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *storage.{{.Type}}) error) error {
	return b.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*storage.{{.Type}}))
	})
}
{{- end}}
