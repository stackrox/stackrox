{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $idx, $pk := .}}{{if $idx}} AND {{end}}{{$pk.ColumnName}} = ${{add $idx 1}}{{end}}{{end}}
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
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
)

const (
        baseTable = "{{.Table}}"
        countStmt = "SELECT COUNT(*) FROM {{.Table}}"
        existsStmt = "SELECT EXISTS(SELECT 1 FROM {{.Table}} WHERE {{template "whereMatch" $pks}})"

        getStmt = "SELECT serialized FROM {{.Table}} WHERE {{template "whereMatch" $pks}}"
        deleteStmt = "DELETE FROM {{.Table}} WHERE {{template "whereMatch" $pks}}"
        walkStmt = "SELECT serialized FROM {{.Table}}"

{{- if $singlePK }}
        getIDsStmt = "SELECT {{$singlePK.ColumnName}} FROM {{.Table}}"
        getManyStmt = "SELECT serialized FROM {{.Table}} WHERE {{$singlePK.ColumnName}} = ANY($1::text[])"

        deleteManyStmt = "DELETE FROM {{.Table}} WHERE {{$singlePK.ColumnName}} = ANY($1::text[])"
{{- end }}
)

func init() {
    globaldb.RegisterTable(baseTable, "{{.TrimmedType}}")
}

type Store interface {
    Count(ctx context.Context) (int, error)
    Exists(ctx context.Context, {{template "paramList" $pks}}) (bool, error)
    Get(ctx context.Context, {{template "paramList" $pks}}) (*{{.Type}}, bool, error)
    Upsert(ctx context.Context, obj *{{.Type}}) error
    UpsertMany(ctx context.Context, objs []*{{.Type}}) error
    Delete(ctx context.Context, {{template "paramList" $pks}}) error

{{- if $singlePK }}
    GetIDs(ctx context.Context) ([]{{$singlePK.Type}}, error)
    GetMany(ctx context.Context, ids []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error)
    DeleteMany(ctx context.Context, ids []{{$singlePK.Type}}) error
{{- end }}

    Walk(ctx context.Context, fn func(obj *{{.Type}}) error) error

    AckKeysIndexed(ctx context.Context, keys ...string) error
    GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
    db *pgxpool.Pool
}

{{- define "createFunctionName"}}createTable{{.Table|upperCamelCase}}
{{- end}}

{{- define "createTable"}}
{{- $schema := . }}
func {{template "createFunctionName" $schema}}(ctx context.Context, db *pgxpool.Pool) {
    table := `
create table if not exists {{$schema.Table}} (
{{- range $idx, $field := $schema.ResolvedFields }}
    {{$field.ColumnName}} {{$field.SQLType}}{{if $field.Options.Unique}} UNIQUE{{end}},
{{- end}}
    PRIMARY KEY({{template "commaSeparatedColumns" $schema.ResolvedPrimaryKeys}}){{ if gt (len $schema.Parents) 0 }},{{end}}
    {{- range $parent, $pks := $schema.ParentKeysAsMap }}
    CONSTRAINT fk_parent_table FOREIGN KEY ({{template "commaSeparatedColumns" $pks}}) REFERENCES {{$parent}}({{template "commandSeparatedRefs" $pks}}) ON DELETE CASCADE
    {{- end }}
)
`

    _, err := db.Exec(ctx, table)
    if err != nil {
        panic("error creating table: " + table)
    }

    indexes := []string {
    {{range $idx, $field := $schema.Fields}}
        {{if $field.Options.Index}}"create index if not exists {{$schema.Table|lowerCamelCase}}_{{$field.ColumnName}} on {{$schema.Table}} using {{$field.Options.Index}}({{$field.ColumnName}})",{{end}}
    {{end}}
    }
    for _, index := range indexes {
       if _, err := db.Exec(ctx, index); err != nil {
           panic(err)
        }
    }

    {{range $idx, $child := $schema.Children}}
    {{template "createFunctionName" $child}}(ctx, db)
    {{- end}}
}
{{range $idx, $child := $schema.Children}}{{template "createTable" $child}}{{end}}
{{end}}
{{- template "createTable" .Schema}}

{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := . }}

func {{ template "insertFunctionName" $schema }}(ctx context.Context, tx pgx.Tx, obj {{$schema.Type}}{{ range $idx, $field := $schema.ParentKeys }}, {{$field.Name}} {{$field.Type}}{{end}}{{if $schema.Parents}}, idx int{{end}}) error {
    {{if not $schema.Parents }}
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }
    {{end}}

    values := []interface{} {
        // parent primary keys start
        {{- range $idx, $field := $schema.ResolvedFields -}}
        {{- if eq $field.DataType "datetime" }}
        pgutils.NilOrStringTimestamp({{$field.Getter "obj"}}),
        {{- else }}
        {{$field.Getter "obj"}},{{end}}
        {{- end}}
    }

    finalStr := "INSERT INTO {{$schema.Table}} ({{template "commaSeparatedColumns" $schema.ResolvedFields }}) VALUES({{ valueExpansion (len $schema.ResolvedFields) }}) ON CONFLICT({{template "commaSeparatedColumns" $schema.ResolvedPrimaryKeys}}) DO UPDATE SET {{template "updateExclusions" $schema.ResolvedFields}}"
    _, err := tx.Exec(ctx, finalStr, values...)
    if err != nil {
        return err
    }

    {{if $schema.Children}}
    var query string
    {{end}}

    {{range $idx, $child := $schema.Children}}
    for childIdx, child := range obj.{{$child.ObjectGetter}} {
        if err := {{ template "insertFunctionName" $child }}(ctx, tx, child{{ range $idx, $field := $schema.ParentKeys }}, {{$field.Name}}{{end}}{{ range $idx, $field := $schema.LocalPrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, childIdx); err != nil {
            return err
        }
    }

    query = "delete from {{$child.Table}} where {{ range $idx, $field := $child.ParentKeys }}{{if $idx}} AND {{end}}{{$field.ColumnName}} = ${{add $idx 1}}{{end}} AND idx >= ${{add (len $child.ParentKeys) 1}}"
    _, err = tx.Exec(ctx, query, {{ range $idx, $field := $schema.ParentKeys }}{{$field.Name}}, {{end}}{{ range $idx, $field := $schema.LocalPrimaryKeys }}{{$field.Getter "obj"}}, {{end}} len(obj.{{$child.ObjectGetter}}))
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
func New(ctx context.Context, db *pgxpool.Pool) Store {
    {{template "createFunctionName" .Schema}}(ctx, db)

    return &storeImpl{
        db: db,
    }
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*{{.Type}}) error {
    conn, release := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
    defer release()

    for _, obj := range objs {
	    tx, err := conn.Begin(ctx)
	    if err != nil {
    		return err
	    }

	    if err := {{ template "insertFunctionName" .Schema }}(ctx, tx, obj); err != nil {
		    if err := tx.Rollback(ctx); err != nil {
			    return err
		    }
		    return err
        }
        if err := tx.Commit(ctx); err != nil {
            return err
        }
    }
    return nil
}

func (s *storeImpl) Upsert(ctx context.Context, obj *{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "{{.TrimmedType}}")

    return s.upsert(ctx, obj)
}

func (s *storeImpl) UpsertMany(ctx context.Context, objs []*{{.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "{{.TrimmedType}}")

    return s.upsert(ctx, objs...)
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "{{.TrimmedType}}")

	row := s.db.QueryRow(ctx, countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, {{template "paramList" $pks}}) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "{{.TrimmedType}}")

	row := s.db.QueryRow(ctx, existsStmt, {{template "argList" $pks}})
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, pgutils.ErrNilIfNoRows(err)
	}
	return exists, nil
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context, {{template "paramList" $pks}}) (*{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.TrimmedType}}")

	conn, release := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
	defer release()

	row := conn.QueryRow(ctx, getStmt, {{template "argList" $pks}})
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

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func()) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		panic(err)
	}
	return conn, conn.Release
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context, {{template "paramList" $pks}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.TrimmedType}}")

	conn, release := s.acquireConn(ctx, ops.Remove, "{{.TrimmedType}}")
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt, {{template "argList" $pks}}); err != nil {
		return err
	}
	return nil
}

{{- if $singlePK }}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]{{$singlePK.Type}}, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "{{.Type}}IDs")

	rows, err := s.db.Query(ctx, getIDsStmt)
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
func (s *storeImpl) GetMany(ctx context.Context, ids []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "{{.TrimmedType}}")

	conn, release := s.acquireConn(ctx, ops.GetMany, "{{.TrimmedType}}")
	defer release()

	rows, err := conn.Query(ctx, getManyStmt, ids)
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
	resultsByID := make(map[{{$singlePK.Type}}]*{{.Type}})
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, err
		}
		msg := &{{.Type}}{}
		if err := proto.Unmarshal(data, msg); err != nil {
		    return nil, nil, err
		}
		resultsByID[{{$singlePK.Getter "msg"}}] = msg
	}
	missingIndices := make([]int, 0, len(ids)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input ids
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*{{.Type}}, 0, len(resultsByID))
	for i, id := range ids {
		if result, ok := resultsByID[id]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
		    elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ctx context.Context, ids []{{$singlePK.Type}}) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "{{.TrimmedType}}")

	conn, release := s.acquireConn(ctx, ops.RemoveMany, "{{.TrimmedType}}")
	defer release()
	if _, err := conn.Exec(ctx, deleteManyStmt, ids); err != nil {
		return err
	}
	return nil
}
{{- end }}

// Walk iterates over all of the objects in the store and applies the closure
func (s *storeImpl) Walk(ctx context.Context, fn func(obj *{{.Type}}) error) error {
	rows, err := s.db.Query(ctx, walkStmt)
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
		if err := fn(&msg); err != nil {
		    return err
		}
	}
	return nil
}

//// Used for testing
{{- define "dropTableFunctionName"}}dropTable{{.Table | upperCamelCase}}{{end}}
{{- define "dropTable"}}
{{- $schema := . }}
func {{ template "dropTableFunctionName" $schema }}(ctx context.Context, db *pgxpool.Pool) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{$schema.Table}} CASCADE")
    {{range $idx, $child := $schema.Children}}{{ template "dropTableFunctionName" $child }}(ctx, db)
    {{end}}
}
{{range $idx, $child := $schema.Children}}{{ template "dropTable" $child }}{{end}}
{{- end}}

{{template "dropTable" .Schema}}

func Destroy(ctx context.Context, db *pgxpool.Pool) {
    {{template "dropTableFunctionName" .Schema}}(ctx, db)
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}
