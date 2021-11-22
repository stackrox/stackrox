package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	_ "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/tools/generate-helpers/pg-table-bindings/walker"
)

const storeFile = `

package postgres

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/set"
)

const (
		countStmt = "select count(*) from {{.Table}}"
		existsStmt = "select exists(select 1 from {{.Table}} where id = $1)"
		getIDsStmt = "select id from {{.Table}}"
		getStmt = "select serialized from {{.Table}} where id = $1"
		getManyStmt = "select serialized from {{.Table}} where id = ANY($1::text[])"
		deleteStmt = "delete from {{.Table}} where id = $1"
		deleteManyStmt = "delete from {{.Table}} where id = ANY($1::text[])"
		walkStmt = "select serialized from {{.Table}}"
		walkWithIDStmt = "select id, serialized from {{.Table}}"
)

var (
	log = logging.LoggerForModule()

	table = "{{.Table}}"

	marshaler = &jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true}
)

type Store interface {
	Count() (int, error)
	Exists(id string) (bool, error)
	GetIDs() ([]string, error)
	Get(id string) (*storage.{{.Type}}, bool, error)
	GetMany(ids []string) ([]*storage.{{.Type}}, []int, error)
	{{- if .NoKeyField}}
	UpsertWithID(id string, obj *storage.{{.Type}}) error
	UpsertManyWithIDs(ids []string, objs []*storage.{{.Type}}) error
	{{- else }}
	Upsert(obj *storage.{{.Type}}) error
	UpsertMany(objs []*storage.{{.Type}}) error
	{{- end}}
	Delete(id string) error
	DeleteMany(ids []string) error
	{{- if .NoKeyField}}
	WalkAllWithID(fn func(id string, obj *storage.{{.Type}}) error) error
	{{- else }}
	Walk(fn func(obj *storage.{{.Type}}) error) error
	{{- end}}
	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}

type storeImpl struct {
	db *pgxpool.Pool
}

func alloc() proto.Message {
	return &storage.{{.Type}}{}
}

{{- if not .NoKeyField}}

func keyFunc(msg proto.Message) string {
	return msg.(*storage.{{.Type}}).{{.KeyFunc}}
}
{{- end}}

{{- if .UniqKeyFunc}}

func uniqKeyFunc(msg proto.Message) string {
	return msg.(*storage.{{.Type}}).{{.UniqKeyFunc}}
}
{{- end}}

const (
	batchInsertTemplate = "{{.BatchInsertionTemplate}}"
)

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool) Store {
	globaldb.RegisterTable(table, "{{.Type}}")

	for _, table := range []string {
		{{range .FlatTableCreationQueries}}"{{.}}",
		{{end}}
	} {
		_, err := db.Exec(context.Background(), table)
		if err != nil {
			panic("error creating table: " + table)
		}
	}

//	{{- if .UniqKeyFunc}}
//	return &storeImpl{
//		crud: generic.NewUniqueKeyCRUD(db, bucket, {{if .NoKeyField}}nil{{else}}keyFunc{{end}}, allocCluster, uniqKeyFunc, {{.TrackIndex}}),
//	}
//	{{- else}}
	return &storeImpl{
		db: db,
	}
//	{{- end}}
}

// Count returns the number of objects in the store
func (s *storeImpl) Count() (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "{{.Type}}")

	row := s.db.QueryRow(context.Background(), countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "{{.Type}}")

	row := s.db.QueryRow(context.Background(), existsStmt, id)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, nilNoRows(err)
	}
	return exists, nil
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs() ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")

	rows, err := s.db.Query(context.Background(), getIDsStmt)
	if err != nil {
		return nil, nilNoRows(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func nilNoRows(err error) error {
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(id string) (*storage.{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.Type}}")

	conn, release := s.acquireConn(ops.Get, "{{.Type}}")
	defer release()

	row := conn.QueryRow(context.Background(), getStmt, id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, nilNoRows(err)
	}

	msg := alloc()
	buf := bytes.NewBuffer(data)
	defer metrics.SetJSONPBOperationDurationTime(time.Now(), "Unmarshal", "{{.Type}}")
	if err := jsonpb.Unmarshal(buf, msg); err != nil {
		return nil, false, err
	}
	return msg.(*storage.{{.Type}}), true, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice 
func (s *storeImpl) GetMany(ids []string) ([]*storage.{{.Type}}, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "{{.Type}}")

	conn, release := s.acquireConn(ops.GetMany, "{{.Type}}")
	defer release()

	rows, err := conn.Query(context.Background(), getManyStmt, ids)
	if err != nil {
		if err == pgx.ErrNoRows {
			missingIndices := make([]int, 0, len(ids))
			for i := range ids {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	defer rows.Close()
	elems := make([]*storage.{{.Type}}, 0, len(ids))
	foundSet := set.NewStringSet()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, err
		}
		msg := alloc()
		buf := bytes.NewBuffer(data)
		t := time.Now()
		if err := jsonpb.Unmarshal(buf, msg); err != nil {
			return nil, nil, err
		}
		metrics.SetJSONPBOperationDurationTime(t, "Unmarshal", "{{.Type}}")
		elem := msg.(*storage.{{.Type}})
		foundSet.Add(elem.GetId())
		elems = append(elems, elem)
	}
	missingIndices := make([]int, 0, len(ids)-len(foundSet))
	for i, id := range ids {
		if !foundSet.Contains(id) {
			missingIndices = append(missingIndices, i)
		}
	}
	return elems, missingIndices, nil
}

{{- if .NoKeyField}}
// UpsertWithID inserts the object into the DB
func (s *storeImpl) UpsertWithID(id string, obj *storage.{{.Type}}) error {
	return upsert(id, obj)
}

// UpsertManyWithIDs batches objects into the DB
func (s *storeImpl) UpsertManyWithIDs(ids []string, objs []*storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")

	// txn? or partial? what is the impact of one not being upserted
	for i, id := range ids {
		if err := s.upsert(id, objs(i)); err != nil {
			return err
		}
	}
	return nil
}
{{- else}}

func convertEnumSliceToIntArray(i interface{}) []int32 {
	enumSlice := reflect.ValueOf(i)
	enumSliceLen := enumSlice.Len()
	resultSlice := make([]int32, 0, enumSliceLen)
	for i := 0; i < enumSlice.Len(); i++ {
		resultSlice = append(resultSlice, int32(enumSlice.Index(i).Int()))
	}
	return resultSlice
}

func nilOrStringTimestamp(t *types.Timestamp) *string {
  if t == nil {
    return nil
  }
  s := t.String()
  return &s
}

func (s *storeImpl) upsert(id string, obj0 *storage.{{.Type}}) error {
	t := time.Now()
	serialized, err := marshaler.MarshalToString(obj0)
	if err != nil {
		return err
	}
	metrics.SetJSONPBOperationDurationTime(t, "Marshal", "{{.Type}}")
	conn, release := s.acquireConn(ops.Add, "{{.Type}}")
	defer release()

	tx, err := conn.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}
    doRollback := true
	defer func() {
		if doRollback {
			if rollbackErr := tx.Rollback(context.Background()); rollbackErr != nil {
				log.Errorf("error rolling backing: %v", err)
			}
		}
	}()

	{{.FlatInsertion}}

    doRollback = false
	return tx.Commit(context.Background())
}

// Upsert inserts the object into the DB
func (s *storeImpl) Upsert(obj *storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Add, "{{.Type}}")
	return s.upsert(keyFunc(obj), obj)
}

func (s *storeImpl) acquireConn(op ops.Op, typ string) (*pgxpool.Conn, func()) {
	defer metrics.SetAcquireDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(context.Background())
	if err != nil {
		panic(err)
	}
	return conn, conn.Release
}

// UpsertMany batches objects into the DB
func (s *storeImpl) UpsertMany(objs []*storage.{{.Type}}) error {
	if len(objs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.AddMany, "{{.Type}}")
	for _, obj0 := range objs {
		t := time.Now()
		serialized, err := marshaler.MarshalToString(obj0)
		if err != nil {
			return err
		}
		metrics.SetJSONPBOperationDurationTime(t, "Marshal", "{{.Type}}")
		{{.FlatMultiInsert}}
	}

	conn, release := s.acquireConn(ops.AddMany, "{{.Type}}")
	defer release()

	results := conn.SendBatch(context.Background(), batch)
	if err := results.Close(); err != nil {
		return err
	}
	return nil
}
{{- end}}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.Type}}")

	conn, release := s.acquireConn(ops.Remove, "{{.Type}}")
	defer release()

	if _, err := conn.Exec(context.Background(), deleteStmt, id); err != nil {
		return err
	}
	return nil
}

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ids []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{.Type}}")

	conn, release := s.acquireConn(ops.RemoveMany, "{{.Type}}")
	defer release()
	if _, err := conn.Exec(context.Background(), deleteManyStmt, ids); err != nil {
		return err
	}
	return nil
}

{{- if .NoKeyField}}
// WalkAllWithID iterates over all of the objects in the store and applies the closure
func (s *storeImpl) WalkAllWithID(fn func(id string, obj *storage.{{.Type}}) error) error {

	panic("unimplemented")	
//return b.crud.WalkAllWithID(func(id []byte, msg proto.Message) error {
	rows, err := s.db.Query(context.Background(), walkStmt)
	if err != nil {
		return nilNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return err
		}
		msg := alloc()
		buf := bytes.NewReader(data)
		if err := jsonpb.Unmarshal(buf, msg); err != nil {
			return err
		}
		return fn(id, msg.(*storage.{{.Type}}))
	}
	return nil
}
{{- else}}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(fn func(obj *storage.{{.Type}}) error) error {
	rows, err := s.db.Query(context.Background(), walkStmt)
	if err != nil {
		return nilNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		msg := alloc()
		buf := bytes.NewReader(data)
		if err := jsonpb.Unmarshal(buf, msg); err != nil {
			return err
		}
		return fn(msg.(*storage.{{.Type}}))
	}
	return nil
}
{{- end}}

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex() ([]string, error) {
	return nil, nil
}
`

type properties struct {
	Type           string
	Table          string
	RegisteredType string
	NoKeyField     bool
	KeyFunc        string
	UniqKeyFunc    string
	Cache          bool
	TrackIndex     bool
}

func main() {
	c := &cobra.Command{
		Use: "generate store implementations",
	}

	var props properties
	c.Flags().StringVar(&props.Type, "type", "", "the (Go) name of the object")
	utils.Must(c.MarkFlagRequired("type"))

	c.Flags().StringVar(&props.RegisteredType, "registered-type", "", "the type this is registered in proto as storage.X")

	c.Flags().StringVar(&props.Table, "table", "", "the logical table of the objects")
	utils.Must(c.MarkFlagRequired("table"))

	c.Flags().BoolVar(&props.NoKeyField, "no-key-field", false, "whether or not object contains key field. If no, then to key function is not applied on object")
	c.Flags().StringVar(&props.KeyFunc, "key-func", "GetId()", "the function on the object to retrieve the key")
	c.Flags().StringVar(&props.UniqKeyFunc, "uniq-key-func", "", "when set, unique key constraint is added on the object field retrieved by the function")

	c.RunE = func(*cobra.Command, []string) error {
		typ := fmt.Sprintf("storage.%s", props.Type)
		if props.RegisteredType != "" {
			typ = fmt.Sprintf("storage.%s", props.RegisteredType)
		}
		fmt.Println("Generating for", typ)
		mt := proto.MessageType(typ)
		props.Table = strings.TrimPrefix(mt.Elem().String(), "storage.")
		table := walker.Walk(mt, props.Table)

		insertion := generateInsertFunctions(table)

		tableCreationQueries := createTables(table)
		var count int
		for _, t := range tableCreationQueries {
			if strings.HasPrefix(t, "create table") {
				count++
			}
		}
		fmt.Println("Number of tables", count)

		t := template.Must(template.New("insertion").Parse(insertion))
		buf := bytes.NewBuffer(nil)
		if err := t.Execute(buf, map[string]interface{} {"ExecutePrefix": "tx.Exec(context.Background(),"}); err != nil {
			return err
		}
		singleInsert := buf.String()

		t = template.Must(template.New("insertion").Parse(insertion))
		buf = bytes.NewBuffer(nil)
		if err := t.Execute(buf, map[string]interface{} {"ExecutePrefix": "batch.Queue("}); err != nil {
			return err
		}
		multiInsert := buf.String()

		templateMap := map[string]interface{}{
			"Type":       props.Type,
			"Bucket":     props.Table,
			"NoKeyField": props.NoKeyField,
			"KeyFunc":    props.KeyFunc,
			//"UniqKeyFunc": props.UniqKeyFunc,
			"Table":                   props.Table,

			"FlatInsertion": singleInsert,
			"FlatTableCreationQueries": tableCreationQueries,
			"FlatMultiInsert": multiInsert,
			"SingleTable": count == 1,
		}

		t = template.Must(template.New("gen").Parse(autogenerated + storeFile))
		buf = bytes.NewBuffer(nil)
		if err := t.Execute(buf, templateMap); err != nil {
			return err
		}
		if err := ioutil.WriteFile("store.go", buf.Bytes(), 0644); err != nil {
			return err
		}
		return nil
	}
	if err := c.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
