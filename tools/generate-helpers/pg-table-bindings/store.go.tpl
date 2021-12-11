{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.Field|lowerCamelCase}} {{$pk.RawFieldType}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.Field|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $idx, $pk := .}}{{if $idx}} AND {{end}}{{$pk.Field}} = ${{add $idx 1}}{{end}}{{end}}

{{- $ := . }}
{{- $pks := $.TopLevelTable.PrimaryKeyElements }}

{{- $singlePK := dict.nil }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{- end }}

package postgres

import (
	"bytes"
	"context"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/stackrox/rox/pkg/set"
)

const (
		countStmt = "SELECT COUNT(*) FROM {{.Table}}"
		existsStmt = "SELECT EXISTS(SELECT 1 FROM {{.Table}} WHERE {{template "whereMatch" $pks}})"

		getStmt = "SELECT serialized FROM {{.Table}} WHERE {{template "whereMatch" $pks}}"
		deleteStmt = "DELETE FROM {{.Table}} WHERE {{template "whereMatch" $pks}}"
		walkStmt = "SELECT serialized FROM {{.Table}}"

{{- if $singlePK }}
        getIDsStmt = "SELECT {{$singlePK.Field}} FROM {{.Table}}"
		getManyStmt = "SELECT serialized FROM {{.Table}} WHERE {{$singlePK.Field}} = ANY($1::text[])"
		deleteManyStmt = "DELETE FROM {{.Table}} WHERE {{$singlePK.Field}} = ANY($1::text[])"
{{- else }}
    {{- range $_, $pk := $pks }}
        deleteBy{{$pk.Field|upperCamelCase}}Stmt = "DELETE FROM {{$.Table}} WHERE {{$pk.Field}} = $1"
    {{- end }}
{{- end }}
)

var (
	log = logging.LoggerForModule()

	table = "{{.Table}}"

	marshaler = &jsonpb.Marshaler{EnumsAsInts: true, EmitDefaults: true}
)

type Store interface {
	Count() (int, error)
	Exists({{template "paramList" $pks}}) (bool, error)
	Get({{template "paramList" $pks}}) (*storage.{{.Type}}, bool, error)
    Upsert(obj *storage.{{.Type}}) error
    UpsertMany(objs []*storage.{{.Type}}) error
    Delete({{template "paramList" $pks}}) error

{{- if $singlePK }}
	GetIDs() ([]{{$singlePK.RawFieldType}}, error)
    GetMany(ids []{{$singlePK.RawFieldType}}) ([]*storage.{{.Type}}, []int, error)
	DeleteMany(ids []{{$singlePK.RawFieldType}}) error
{{- else }}
{{- range $_, $pk := $pks }}
	DeleteBy{{$pk.Field|upperCamelCase}}({{$pk.Field|lowerCamelCase}} {{$pk.RawFieldType}}) error
{{- end }}
{{- end }}

	Walk(fn func(obj *storage.{{.Type}}) error) error
	AckKeysIndexed(keys ...string) error
	GetKeysToIndex() ([]string, error)
}

type storeImpl struct {
	db *pgxpool.Pool
}

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
func (s *storeImpl) Exists({{template "paramList" $pks}}) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "{{.Type}}")

	row := s.db.QueryRow(context.Background(), existsStmt, {{template "argList" $pks}})
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, nilNoRows(err)
	}
	return exists, nil
}

func nilNoRows(err error) error {
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get({{template "paramList" $pks}}) (*storage.{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.Type}}")

	conn, release := s.acquireConn(ops.Get, "{{.Type}}")
	defer release()

	row := conn.QueryRow(context.Background(), getStmt, {{template "argList" $pks}})
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, nilNoRows(err)
	}

	var msg storage.{{.Type}}
	buf := bytes.NewBuffer(data)
	defer metrics.SetJSONPBOperationDurationTime(time.Now(), "Unmarshal", "{{.Type}}")
	if err := jsonpb.Unmarshal(buf, &msg); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

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

// Upsert inserts the object into the DB
func (s *storeImpl) Upsert(obj0 *storage.{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Add, "{{.Type}}")

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


// Delete removes the specified ID from the store
func (s *storeImpl) Delete({{template "paramList" $pks}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.Type}}")

	conn, release := s.acquireConn(ops.Remove, "{{.Type}}")
	defer release()

	if _, err := conn.Exec(context.Background(), deleteStmt, {{template "argList" $pks}}); err != nil {
		return err
	}
	return nil
}

{{- if $singlePK }}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs() ([]{{$singlePK.RawFieldType}}, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")

	rows, err := s.db.Query(context.Background(), getIDsStmt)
	if err != nil {
		return nil, nilNoRows(err)
	}
	defer rows.Close()
	var ids []{{$singlePK.RawFieldType}}
	for rows.Next() {
		var id {{$singlePK.RawFieldType}}
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ids []{{$singlePK.RawFieldType}}) ([]*storage.{{.Type}}, []int, error) {
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
	foundSet := make(map[{{$singlePK.RawFieldType}}]struct{})
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, err
		}
		var msg storage.{{.Type}}
		buf := bytes.NewBuffer(data)
		t := time.Now()
		if err := jsonpb.Unmarshal(buf, &msg); err != nil {
			return nil, nil, err
		}
		metrics.SetJSONPBOperationDurationTime(t, "Unmarshal", "{{.Type}}")
		foundSet[msg.Get{{$singlePK.Field|upperCamelCase}}()] = struct{}{}
		elems = append(elems, &msg)
	}
	missingIndices := make([]int, 0, len(ids)-len(foundSet))
	for i, id := range ids {
		if _, ok := foundSet[id]; !ok {
			missingIndices = append(missingIndices, i)
		}
	}
	return elems, missingIndices, nil
}

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ids []{{$singlePK.RawFieldType}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{.Type}}")

	conn, release := s.acquireConn(ops.RemoveMany, "{{.Type}}")
	defer release()
	if _, err := conn.Exec(context.Background(), deleteManyStmt, ids); err != nil {
		return err
	}
	return nil
}

{{- else }}
{{- range $_, $pk := $pks }}
func (s *storeImpl) DeleteBy{{$pk.Field|upperCamelCase}}({{$pk.Field|lowerCamelCase}} {{$pk.RawFieldType}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{$.Type}}")

	conn, release := s.acquireConn(ops.RemoveMany, "{{$.Type}}")
	defer release()
	if _, err := conn.Exec(context.Background(), deleteBy{{$pk.Field|upperCamelCase}}Stmt, {{$pk.Field|lowerCamelCase}}); err != nil {
		return err
	}
	return nil
}
{{- end }}
{{- end }}

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
		var msg storage.{{.Type}}
		buf := bytes.NewReader(data)
		if err := jsonpb.Unmarshal(buf, &msg); err != nil {
			return err
		}
		return fn(&msg)
	}
	return nil
}

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex() ([]string, error) {
	return nil, nil
}
