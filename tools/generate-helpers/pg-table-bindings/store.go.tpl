{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.Name|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.Name|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $idx, $pk := .}}{{if $idx}} AND {{end}}{{$pk.Name}} = ${{add $idx 1}}{{end}}{{end}}
{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}

{{- $ := . }}
{{- $pks := .Schema.LocalPrimaryKeys }}

{{- $singlePK := dict.nil }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{- end }}

package postgres

import (
    "context"
    "fmt"
    "time"

    "github.com/gogo/protobuf/proto"
    "github.com/jackc/pgx/v4/pgxpool"
    "github.com/jackc/pgx/v4"
    "github.com/stackrox/rox/central/globaldb"
    "github.com/stackrox/rox/central/metrics"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
)

const (
        countStmt = "SELECT COUNT(*) FROM {{.Table}}"
        existsStmt = "SELECT EXISTS(SELECT 1 FROM {{.Table}} WHERE {{template "whereMatch" $pks}})"

        getStmt = "SELECT serialized FROM {{.Table}} WHERE {{template "whereMatch" $pks}}"
        deleteStmt = "DELETE FROM {{.Table}} WHERE {{template "whereMatch" $pks}}"
        walkStmt = "SELECT serialized FROM {{.Table}}"

{{- if $singlePK }}
        getIDsStmt = "SELECT {{$singlePK.Name}} FROM {{.Table}}"
        getManyStmt = "SELECT serialized FROM {{.Table}} WHERE {{$singlePK.Name}} = ANY($1::text[])"

        deleteManyStmt = "DELETE FROM {{.Table}} WHERE {{$singlePK.Name}} = ANY($1::text[])"
{{- end }}
)

var (
    log = logging.LoggerForModule()

    table = "{{.Table}}"
)

type Store interface {
    Count() (int, error)
    Exists({{template "paramList" $pks}}) (bool, error)
    Get({{template "paramList" $pks}}) (*{{.Type}}, bool, error)
    Upsert(obj *{{.Type}}) error
    UpsertMany(objs []*{{.Type}}) error
    Delete({{template "paramList" $pks}}) error

{{- if $singlePK }}
    GetIDs() ([]{{$singlePK.Type}}, error)
    GetMany(ids []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error)
    DeleteMany(ids []{{$singlePK.Type}}) error
{{- end }}

    Walk(fn func(obj *{{.Type}}) error) error
    AckKeysIndexed(keys ...string) error
    GetKeysToIndex() ([]string, error)
}

type storeImpl struct {
    db *pgxpool.Pool
}

{{- define "createFunctionName"}}createTable{{.Table|upperCamelCase}}
{{- end}}

{{- define "createTable"}}
{{- $schema := . }}
func {{template "createFunctionName" $schema}}(db *pgxpool.Pool) {
    // hack for testing, remove
    db.Exec(context.Background(), "DROP TABLE {{$schema.Table}} CASCADE")

    table := `
create table if not exists {{$schema.Table}} (
{{- range $idx, $field := $schema.ResolvedFields }}
    {{$field.ColumnName}} {{$field.SQLType}}{{if $field.Options.Unique}} UNIQUE{{end}},
{{- end}}
    PRIMARY KEY({{template "commaSeparatedColumns" $schema.ResolvedPrimaryKeys}}){{ if $schema.ParentSchema }},
    {{- $pks := $schema.ParentKeys }}
    CONSTRAINT fk_parent_table FOREIGN KEY ({{template "commaSeparatedColumns" $pks}}) REFERENCES {{$schema.ParentSchema.Table}}({{template "commandSeparatedRefs" $pks}}) ON DELETE CASCADE
    {{- end }}
)
`

    _, err := db.Exec(context.Background(), table)
    if err != nil {
        panic("error creating table: " + table)
    }

    indexes := []string {
    {{range $idx, $field := $schema.Fields}}
        {{if $field.Options.Index}}"create index if not exists {{$schema.Table|lowerCamelCase}}_{{$field.ColumnName}} on {{$schema.Table}} using {{$field.Options.Index}}({{$field.ColumnName}})",{{end}}
    {{end}}
    }
    for _, index := range indexes {
       if _, err := db.Exec(context.Background(), index); err != nil {
           panic(err)
        }
    }

    {{range $idx, $child := $schema.Children}}
    {{template "createFunctionName" $child}}(db)
    {{- end}}
}
{{range $idx, $child := $schema.Children}}{{template "createTable" $child}}{{end}}
{{end}}
{{- template "createTable" .Schema}}

{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := . }}

func {{ template "insertFunctionName" $schema }}(db *pgxpool.Pool, obj {{$schema.Type}}{{ range $idx, $field := $schema.ParentKeys }}, {{$field.Name}} {{$field.Type}}{{end}}{{if $schema.ParentSchema}}, idx int{{end}}) error {
    {{if not $schema.ParentSchema }}
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }
    {{end}}

    values := []interface{} {
        // parent primary keys start
        {{ range $idx, $field := $schema.ResolvedFields }}
        {{$field.Getter "obj"}},{{end}}
    }

    finalStr := "INSERT INTO {{$schema.Table}} ({{template "commaSeparatedColumns" $schema.ResolvedFields }}) VALUES({{ valueExpansion (len $schema.ResolvedFields) }}) ON CONFLICT({{template "commaSeparatedColumns" $schema.ResolvedPrimaryKeys}}) DO UPDATE SET {{template "updateExclusions" $schema.ResolvedFields}}"
    _, err := db.Exec(context.Background(), finalStr, values...)
    if err != nil {
        return err
    }

    {{if $schema.Children}}
    var query string
    {{end}}

    {{range $idx, $child := $schema.Children}}
    for childIdx, child := range obj.{{$child.ObjectGetter}} {
        if err := {{ template "insertFunctionName" $child }}(db, child{{ range $idx, $field := $schema.ParentKeys }}, {{$field.Name}}{{end}}{{ range $idx, $field := $schema.LocalPrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, childIdx); err != nil {
            return err
        }
    }

    query = "delete from {{$child.Table}} where {{ range $idx, $field := $child.ParentKeys }}{{if $idx}} AND {{end}}{{$field.ColumnName}} = ${{add $idx 1}}{{end}} AND idx >= ${{add (len $child.ParentKeys) 1}}"
    _, err = db.Exec(context.Background(), query, {{ range $idx, $field := $schema.ParentKeys }}{{$field.Name}}, {{end}}{{ range $idx, $field := $schema.LocalPrimaryKeys }}{{$field.Getter "obj"}}, {{end}} len(obj.{{$child.ObjectGetter}}))
    if err != nil {
        return err
    }

    {{- end}}
    return nil
}
{{range $idx, $child := $schema.Children}}{{ template "insertObject" $child }}{{end}}
{{- end}}

{{ template "insertObject" .Schema }}

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool) Store {
    globaldb.RegisterTable(table, "{{.Type}}")

    {{template "createFunctionName" .Schema}}(db)

    return &storeImpl{
        db: db,
    }
}

func (s *storeImpl) Upsert(obj *{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "{{.Type}}")

	return {{ template "insertFunctionName" .Schema }}(s.db, obj)
}

func (s *storeImpl) UpsertMany(objs []*{{.Type}}) error {
	for _, obj := range objs {
		if err := {{ template "insertFunctionName" .Schema }}(s.db, obj); err != nil {
			return err
		}
	}
	return nil
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
		return false, pgutils.ErrNilIfNoRows(err)
	}
	return exists, nil
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get({{template "paramList" $pks}}) (*{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.Type}}")

	conn, release := s.acquireConn(ops.Get, "{{.Type}}")
	defer release()

	row := conn.QueryRow(context.Background(), getStmt, {{template "argList" $pks}})
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg {{.Type}}
	if err := proto.Unmarshal(data, &msg); err != nil {
        return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) acquireConn(op ops.Op, typ string) (*pgxpool.Conn, func()) {
	defer metrics.SetAcquireDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(context.Background())
	if err != nil {
		panic(err)
	}
	return conn, conn.Release
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
func (s *storeImpl) GetIDs() ([]{{$singlePK.Type}}, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")

	rows, err := s.db.Query(context.Background(), getIDsStmt)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	var ids []{{$singlePK.Type}}
	for rows.Next() {
		var id {{$singlePK.Type}}
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ids []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error) {
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
	elems := make([]*{{.Type}}, 0, len(ids))
	foundSet := make(map[{{$singlePK.Type}}]struct{})
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, err
		}
		var msg {{.Type}}
		if err := proto.Unmarshal(data, &msg); err != nil {
		    return nil, nil, err
		}
		foundSet[{{$singlePK.Getter "msg"}}] = struct{}{}
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
func (s *storeImpl) DeleteMany(ids []{{$singlePK.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{.Type}}")

	conn, release := s.acquireConn(ops.RemoveMany, "{{.Type}}")
	defer release()
	if _, err := conn.Exec(context.Background(), deleteManyStmt, ids); err != nil {
		return err
	}
	return nil
}
{{- end }}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(fn func(obj *{{.Type}}) error) error {
	rows, err := s.db.Query(context.Background(), walkStmt)
	if err != nil {
		return pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return err
		}
		var msg {{.Type}}
		if err := proto.Unmarshal(data, &msg); err != nil {
		    return err
		}
		return fn(&msg)
	}
	return nil
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex() ([]string, error) {
	return nil, nil
}
