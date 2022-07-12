package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
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
	{{- if .NoKeyField}}
	UpsertWithID(ctx context.Context, id string, obj *storage.{{.Type}}) error
	UpsertManyWithIDs(ctx context.Context, ids []string, objs []*storage.{{.Type}}) error
	{{- else }}
	Upsert(ctx context.Context, obj *storage.{{.Type}}) error
	UpsertMany(ctx context.Context, objs []*storage.{{.Type}}) error
	{{- end}}
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	{{- if .NoKeyField}}
	WalkAllWithID(ctx context.Context, fn func(id string, obj *storage.{{.Type}}) error) error
	{{- else }}
	Walk(ctx context.Context, fn func(obj *storage.{{.Type}}) error) error
	{{- end}}
	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
	crud db.Crud
}

func alloc() proto.Message {
	return &storage.{{.Type}}{}
}

{{- if not .NoKeyField}}

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
{{- if .Cache}}
func New(db *rocksdb.RocksDB) (Store, error) {
	globaldb.RegisterBucket(bucket, "{{.Type}}")
	{{- if .UniqKeyFunc}}
	baseCRUD := generic.NewUniqueKeyCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, alloc, uniqKeyFunc, {{.TrackIndex}})
	{{- else}}
	baseCRUD := generic.NewCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, alloc, {{.TrackIndex}})
	{{- end}}
	cacheCRUD, err := mapcache.NewMapCache(baseCRUD, {{if .NoKeyField}}nil{{else}}keyFunc{{end}})
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
		crud: generic.NewUniqueKeyCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, alloc, uniqKeyFunc, {{.TrackIndex}}),
	}
	{{- else}}
	return &storeImpl{
		crud: generic.NewCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, alloc, {{.TrackIndex}}),
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

{{- if .NoKeyField}}
// UpsertWithID inserts the object into the DB
func (b *storeImpl) UpsertWithID(_ context.Context, id string, obj *storage.{{.Type}}) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.Add, "{{.Type}}")

	return b.crud.UpsertWithID(id, obj)
}

// UpsertManyWithIDs batches objects into the DB
func (b *storeImpl) UpsertManyWithIDs(_ context.Context, ids []string, objs []*storage.{{.Type}}) error {
	defer metrics.SetRocksDBOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")

	msgs := make([]proto.Message, 0, len(objs))
	for _, o := range objs {
		msgs = append(msgs, o)
    }

	return b.crud.UpsertManyWithIDs(ids, msgs)
}
{{- else}}

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
{{- end}}

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

// AckKeysIndexed acknowledges the passed keys were indexed
func (b *storeImpl) AckKeysIndexed(_ context.Context, keys ...string) error {
	return b.crud.AckKeysIndexed(keys...)
}

// GetKeysToIndex returns the keys that need to be indexed
func (b *storeImpl) GetKeysToIndex(_ context.Context) ([]string, error) {
	return b.crud.GetKeysToIndex()
}
`

//go:embed rocksdb_plugin.go.tpl
var rocksdbPluginFile string

type properties struct {
	Type        string
	Bucket      string
	NoKeyField  bool
	KeyFunc     string
	UniqKeyFunc string
	Cache       bool
	TrackIndex  bool
	// Migration root
	MigrationRoot string
	// The unique sequence number to migrate to Postgres
	MigrationSeq int
	// Where the data are migrated from in the format of "database:bucket", eg, \"rocksdb:alerts\" or \"boltdb:version\"")
	MigrateToTable string
}

type migrationOptions struct {
	Package string
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

	c.Flags().BoolVar(&props.NoKeyField, "no-key-field", false, "whether or not object contains key field. If no, then to key function is not applied on object")
	c.Flags().StringVar(&props.KeyFunc, "key-func", "GetId()", "the function on the object to retrieve the key")
	c.Flags().StringVar(&props.UniqKeyFunc, "uniq-key-func", "", "when set, unique key constraint is added on the object field retrieved by the function")
	c.Flags().BoolVar(&props.Cache, "cache", false, "whether or not to add a fully inmem cache")
	c.Flags().BoolVar(&props.TrackIndex, "track-index", false, "whether or not to track the index updates and wait for them to be acknowledged")
	c.Flags().StringVar(&props.MigrationRoot, "migration-root", "", "Root for migrations")
	c.Flags().StringVar(&props.MigrateToTable, "migrate-to", "", "where the data are migrated from in the format of \"<database>:<bucket>\", eg, \"rocksdb:alerts\" or \"boltdb:version\"")
	c.Flags().IntVar(&props.MigrationSeq, "migration-seq", 0, "the unique sequence number to migrate to Postgres")

	c.RunE = func(*cobra.Command, []string) error {
		templateMap := map[string]interface{}{
			"Type":        props.Type,
			"Bucket":      props.Bucket,
			"NoKeyField":  props.NoKeyField,
			"KeyFunc":     props.KeyFunc,
			"UniqKeyFunc": props.UniqKeyFunc,
			"Cache":       props.Cache,
			"TrackIndex":  props.TrackIndex,
		}
		migrationDir := fmt.Sprintf("n_%02d_to_n_%02d_postgres_%s", props.MigrationSeq, props.MigrationSeq+1, props.MigrateToTable)
		root := filepath.Join(props.MigrationRoot, migrationDir)

		t := template.Must(template.New("gen").Parse(autogenerated + storeFile))
		buf := bytes.NewBuffer(nil)
		if err := t.Execute(buf, templateMap); err != nil {
			return err
		}
		if err := os.WriteFile("store.go", buf.Bytes(), 0644); err != nil {
			return err
		}
		if props.MigrationSeq == 0 {
			return nil
		}
		buf.Truncate(0)
		templateMap["Migration"] = migrationOptions{
			Package: fmt.Sprintf("n%dton%d", props.MigrationSeq, props.MigrationSeq+1),
		}

		t = template.Must(template.New("gen").Parse(autogenerated + rocksdbPluginFile))
		buf = bytes.NewBuffer(nil)
		if err := t.Execute(buf, templateMap); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, "legacy/rocksdb_plugin.go"), buf.Bytes(), 0644); err != nil {
			return err
		}

		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
