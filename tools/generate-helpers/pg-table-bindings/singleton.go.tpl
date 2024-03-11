{{define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{define "paramList"}}{{range $index, $pk := .}}{{if $index}}, {{end}}{{$pk.ColumnName|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $index, $pk := .}}{{if $index}}, {{end}}{{$pk.ColumnName|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $index, $pk := .}}{{if $index}} AND {{end}}{{$pk.ColumnName}} = ${{add $index 1}}{{end}}{{end}}
{{define "commaSeparatedColumns"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}

{{- $ := . }}

import (
    "context"
    "strings"
    "time"

    "github.com/stackrox/rox/pkg/postgres"
    "github.com/pkg/errors"
    "github.com/stackrox/rox/central/metrics"
    pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/auth/permissions"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    "github.com/stackrox/rox/pkg/protocompat"
    "github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
    "github.com/stackrox/rox/pkg/search"
    pgSearch "github.com/stackrox/rox/pkg/search/postgres"
    "github.com/stackrox/rox/pkg/sync"
    "github.com/stackrox/rox/pkg/uuid"
)

const (
        baseTable = "{{.Table}}"

        getStmt = "SELECT serialized FROM {{.Table}} LIMIT 1"
        deleteStmt = "DELETE FROM {{.Table}}"
)

var (
    log = logging.LoggerForModule()
    schema = {{ template "schemaVar" .Schema}}
    {{ if and (or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped)) -}}
    targetResource = resources.{{.Type | storageToResource}}
    {{- end }}
)

// Store is the interface to interact with the storage for {{.Type}}
type Store interface {
    Get(ctx context.Context) (*{{.Type}}, bool, error)
    Upsert(ctx context.Context, obj *{{.Type}}) error
    Delete(ctx context.Context) error
}

type storeImpl struct {
    db postgres.DB
    mutex sync.Mutex
}

{{ define "defineScopeChecker" }}scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_{{ . }}_ACCESS).Resource(targetResource){{ end }}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
    return &storeImpl{
        db: db,
    }
}

{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := .schema }}
func {{ template "insertFunctionName" $schema }}(ctx context.Context, tx *postgres.Tx, obj {{$schema.Type}}{{ range $field := $schema.FieldsDeterminedByParent }}, {{$field.Name}} {{$field.Type}}{{end}}) error {
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }

    values := []interface{} {
        // parent primary keys start
        {{- range $field := $schema.DBColumnFields -}}
        {{- if eq $field.DataType "datetime" }}
        protocompat.NilOrTime({{$field.Getter "obj"}}),
        {{- else if eq $field.SQLType "uuid" }}
        pgutils.NilOrUUID({{$field.Getter "obj"}}),
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

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *{{.Type}}) error {
    defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }

    return pgutils.Retry(func() error {
        return s.retryableUpsert(ctx, obj)
    })
}

func (s *storeImpl) retryableUpsert(ctx context.Context, obj *{{.Type}}) error {
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

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context) (*{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, false, nil
    }

    return pgutils.Retry3(func()(*{{.Type}}, bool, error) {
        return s.retryableGet(ctx)
    })
}

func (s *storeImpl) retryableGet(ctx context.Context) (*{{.Type}}, bool, error) {
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
	if err := msg.Unmarshal(data); err != nil {
        return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
	    return nil, nil, err
	}
	return conn, conn.Release, nil
}

// Delete removes the singleton from the store
func (s *storeImpl) Delete(ctx context.Context) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }

    return pgutils.Retry(func() error {
        return s.retryableDelete(ctx)
    })
}

func (s *storeImpl) retryableDelete(ctx context.Context) error {
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

// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db postgres.DB) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{.Schema.Table}} CASCADE")
}
