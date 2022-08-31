{{ $inMigration := ne (index . "Migration") nil}}
{{define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{define "paramList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $idx, $pk := .}}{{if $idx}}, {{end}}{{$pk.ColumnName|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $idx, $pk := .}}{{if $idx}} AND {{end}}{{$pk.ColumnName}} = ${{add $idx 1}}{{end}}{{end}}
{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}

{{- $ := . }}

import (
    "context"
    "strings"
    "time"

    "github.com/gogo/protobuf/proto"
    "github.com/jackc/pgx/v4"
    "github.com/jackc/pgx/v4/pgxpool"
    "github.com/pkg/errors"
    {{- if not $inMigration}}
    "github.com/stackrox/rox/central/metrics"
    "github.com/stackrox/rox/central/role/resources"
    {{- end}}
    pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/auth/permissions"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    "github.com/stackrox/rox/pkg/sac"
    "github.com/stackrox/rox/pkg/search"
    "github.com/stackrox/rox/pkg/search/postgres"
    "github.com/stackrox/rox/pkg/sync"
)

const (
        baseTable = "{{.Table}}"

        getStmt = "SELECT serialized FROM {{.Table}} LIMIT 1"
        deleteStmt = "DELETE FROM {{.Table}}"
)

var (
    log = logging.LoggerForModule()
    schema = {{ template "schemaVar" .Schema}}
    {{ if and (not $inMigration) (or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped)) -}}
    targetResource = resources.{{.Type | storageToResource}}
    {{- end }}
)

type Store interface {
    Get(ctx context.Context) (*{{.Type}}, bool, error)
    Upsert(ctx context.Context, obj *{{.Type}}) error
    Delete(ctx context.Context) error
}

type storeImpl struct {
    db *pgxpool.Pool
    mutex sync.Mutex
}

{{ define "defineScopeChecker" }}scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_{{ . }}_ACCESS).Resource(targetResource){{ end }}

{{define "createTableStmtVar"}}pkgSchema.CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{- define "createTable" }}
{{- $schema := . }}
pgutils.CreateTable(ctx, db, {{template "createTableStmtVar" $schema}})
{{- end }}

// New returns a new Store instance using the provided sql instance.
func New(ctx context.Context, db *pgxpool.Pool) Store {
    {{- range $reference := .Schema.References }}
    {{- template "createTable" $reference.OtherSchema }}
    {{- end }}
    {{- template "createTable" .Schema}}

    return &storeImpl{
        db: db,
    }
}

{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := .schema }}
func {{ template "insertFunctionName" $schema }}(ctx context.Context, tx pgx.Tx, obj {{$schema.Type}}{{ range $field := $schema.FieldsDeterminedByParent }}, {{$field.Name}} {{$field.Type}}{{end}}) error {
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }

    values := []interface{} {
        // parent primary keys start
        {{- range $field := $schema.DBColumnFields -}}
        {{- if eq $field.DataType "datetime" }}
        pgutils.NilOrTime({{$field.Getter "obj"}}),
        {{- else }}
        {{$field.Getter "obj"}},{{end}}
        {{- end}}
    }

    finalStr := "INSERT INTO {{$schema.Table}} ({{template "commaSeparatedColumns" $schema.DBColumnFields }}) VALUES({{ valueExpansion (len $schema.DBColumnFields) }})"
    _, err := tx.Exec(ctx, finalStr, values...)
    if err != nil {
        return err
    }
    return nil
}
{{- end}}

{{ template "insertObject" dict "schema" .Schema "joinTable" .JoinTable }}

func (s *storeImpl) Upsert(ctx context.Context, obj *{{.Type}}) error {
    {{- if not $inMigration}}
    defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{ end }}
    conn, release, err := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
	if err != nil {
	    return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
        return err
	}

    if _, err := tx.Exec(ctx, deleteStmt); err != nil {
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
    return nil
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context) (*{{.Type}}, bool, error) {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, false, nil
    }
    {{ end}}
	conn, release, err := s.acquireConn(ctx, ops.Get, "{{.TrimmedType}}")
	if err != nil {
	    return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getStmt)
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

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
    {{- if not $inMigration}}
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
    {{- end}}
	conn, err := s.db.Acquire(ctx)
	if err != nil {
	    return nil, nil, err
	}
	return conn, conn.Release, nil
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context) error {
    {{- if not $inMigration}}
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{ end}}
    conn, release, err := s.acquireConn(ctx, ops.Remove, "{{.TrimmedType}}")
	if err != nil {
	    return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt); err != nil {
		return err
	}
	return nil
}

// Used for Testing

func Destroy(ctx context.Context, db *pgxpool.Pool) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{.Schema.Table}} CASCADE")
}
