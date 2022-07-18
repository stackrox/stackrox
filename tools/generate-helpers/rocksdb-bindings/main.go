package main

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/utils"
)

const storeFile = `

package rocksdb

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/db"
{{- if .Cache }}
	"github.com/stackrox/rox/pkg/db/mapcache"
{{- end }}
	"github.com/stackrox/rox/pkg/rocksdb"
	generic "github.com/stackrox/rox/pkg/rocksdb/crud"
)

var (
	log = logging.LoggerForModule()

	bucket = []byte("{{.Bucket}}")
)

type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	GetIDs(ctx context.Context) ([]string, error)
	Get(ctx context.Context, id string) (*storage.{{.Type}}, bool, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.{{.Type}}, []int, error)
	Upsert(ctx context.Context, obj *storage.{{.Type}}) error
	UpsertMany(ctx context.Context, objs []*storage.{{.Type}}) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	Walk(ctx context.Context, fn func(obj *storage.{{.Type}}) error) error
	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
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
{{- if .Cache}}
func New(db *rocksdb.RocksDB) (Store, error) {
	globaldb.RegisterBucket(bucket, "{{.Type}}")
	{{- if .UniqKeyFunc}}
	baseCRUD := generic.NewUniqueKeyCRUD(db, bucket, keyFunc, alloc, uniqKeyFunc, {{.TrackIndex}})
	{{- else}}
	baseCRUD := generic.NewCRUD(db, bucket, keyFunc, alloc, {{.TrackIndex}})
	{{- end}}
	cacheCRUD, err := mapcache.NewMapCache(baseCRUD, keyFunc)
	if err != nil {
		return nil, err
	}
	return &storeImpl{
		crud: cacheCRUD,
	}, nil
}
{{- else}}
func New(db *rocksdb.RocksDB) Store {
	globaldb.RegisterBucket(bucket, "{{.Type}}")
	{{- if .UniqKeyFunc}}
	return &storeImpl{
		crud: generic.NewUniqueKeyCRUD(db, bucket, keyFunc, alloc, uniqKeyFunc, {{.TrackIndex}}),
	}
	{{- else}}
	return &storeImpl{
		crud: generic.NewCRUD(db, bucket, keyFunc, alloc, {{.TrackIndex}}),
	}
	{{- end}}
}
{{- end}}

// Count returns the number of objects in the store
func (b *storeImpl) Count(_ context.Context) (int, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Count, "{{.Type}}")

	return b.crud.Count()
}

// Exists returns if the id exists in the store
func (b *storeImpl) Exists(_ context.Context, id string) (bool, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Exists, "{{.Type}}")

	return b.crud.Exists(id)
}

// GetIDs returns all the IDs for the store
func (b *storeImpl) GetIDs(_ context.Context) ([]string, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")

	return b.crud.GetKeys()
}

// Get returns the object, if it exists from the store
func (b *storeImpl) Get(_ context.Context, id string) (*storage.{{.Type}}, bool, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Get, "{{.Type}}")

	msg, exists, err := b.crud.Get(id)
	if err != nil || !exists {
		return nil, false, err
	}
	return msg.(*storage.{{.Type}}), true, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (b *storeImpl) GetMany(_ context.Context, ids []string) ([]*storage.{{.Type}}, []int, error) {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.GetMany, "{{.Type}}")

	msgs, missingIndices, err := b.crud.GetMany(ids)
	if err != nil {
		return nil, nil, err
	}
	objs := make([]*storage.{{.Type}}, 0, len(msgs))
	for _, m := range msgs {
		objs = append(objs, m.(*storage.{{.Type}}))
	}
	return objs, missingIndices, nil
}

// Upsert inserts the object into the DB
func (b *storeImpl) Upsert(_ context.Context, obj *storage.{{.Type}}) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Add, "{{.Type}}")

	return b.crud.Upsert(obj)
}

// UpsertMany batches objects into the DB
func (b *storeImpl) UpsertMany(_ context.Context, objs []*storage.{{.Type}}) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")

	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
    }

	return b.crud.UpsertMany(msgs)
}

// Delete removes the specified ID from the store
func (b *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Remove, "{{.Type}}")

	return b.crud.Delete(id)
}

// Delete removes the specified IDs from the store
func (b *storeImpl) DeleteMany(_ context.Context, ids []string) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.RemoveMany, "{{.Type}}")

	return b.crud.DeleteMany(ids)
}

// Walk iterates over all of the objects in the store and applies the closure
func (b *storeImpl) Walk(_ context.Context, fn func(obj *storage.{{.Type}}) error) error {
	return b.crud.Walk(func(msg proto.Message) error {
		return fn(msg.(*storage.{{.Type}}))
	})
}

// AckKeysIndexed acknowledges the passed keys were indexed
func (b *storeImpl) AckKeysIndexed(_ context.Context, keys ...string) error {
	return b.crud.AckKeysIndexed(keys...)
}

// GetKeysToIndex returns the keys that need to be indexed
func (b *storeImpl) GetKeysToIndex(_ context.Context) ([]string, error) {
	return b.crud.GetKeysToIndex()
}
`

type properties struct {
	Type        string
	Bucket      string
	KeyFunc     string
	UniqKeyFunc string
	Cache       bool
	TrackIndex  bool
}

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	var props properties
	c.Flags().StringVar(&props.Type, "type", "", "the (Go) name of the object")
	utils.Must(c.MarkFlagRequired("type"))

	c.Flags().StringVar(&props.Bucket, "bucket", "", "the logical bucket of the objects")
	utils.Must(c.MarkFlagRequired("bucket"))

	c.Flags().StringVar(&props.KeyFunc, "key-func", "GetId()", "the function on the object to retrieve the key")
	c.Flags().StringVar(&props.UniqKeyFunc, "uniq-key-func", "", "when set, unique key constraint is added on the object field retrieved by the function")
	c.Flags().BoolVar(&props.Cache, "cache", false, "whether or not to add a fully inmem cache")
	c.Flags().BoolVar(&props.TrackIndex, "track-index", false, "whether or not to track the index updates and wait for them to be acknowledged")

	c.RunE = func(*cobra.Command, []string) error {
		templateMap := map[string]interface{}{
			"Type":        props.Type,
			"Bucket":      props.Bucket,
			"KeyFunc":     props.KeyFunc,
			"UniqKeyFunc": props.UniqKeyFunc,
			"Cache":       props.Cache,
			"TrackIndex":  props.TrackIndex,
		}

		t := template.Must(template.New("gen").Parse(autogenerated + storeFile))
		buf := bytes.NewBuffer(nil)
		if err := t.Execute(buf, templateMap); err != nil {
			return err
		}
		if err := os.WriteFile("store.go", buf.Bytes(), 0644); err != nil {
			return err
		}
		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
