{{define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{define "commaSeparatedColumns"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}

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
    "google.golang.org/protobuf/encoding/protojson"
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
    targetResource = resources.{{.ScopingResource}}
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
    serialized, marshalErr := protojson.Marshal(obj)
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

    return pgutils.Retry(ctx, func() error {
        return s.retryableUpsert(ctx, obj)
    })
}

func (s *storeImpl) retryableUpsert(ctx context.Context, obj *{{.Type}}) error {
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteStmt); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
		}
		return errors.Wrap(err, "deleting from {{.Table}}")
	}

	if err := {{ template "insertFunctionName" .Schema }}(ctx, tx, obj); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
		}
		return errors.Wrap(err, "inserting into {{.Table}}")
	}

	return tx.Commit(ctx)
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context) (*{{.Type}}, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ" }}
    if !scopeChecker.IsAllowed() {
        return nil, false, nil
    }

    return pgutils.Retry3(ctx, func()(*{{.Type}}, bool, error) {
        return s.retryableGet(ctx)
    })
}

func (s *storeImpl) retryableGet(ctx context.Context) (*{{.Type}}, bool, error) {
	row := s.db.QueryRow(ctx, getStmt)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg {{.Type}}
	if err := protojson.Unmarshal(data, &msg); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) begin(ctx context.Context) (*postgres.Tx, context.Context, error) {
	return postgres.GetTransaction(ctx, s.db)
}

// Delete removes the singleton from the store
func (s *storeImpl) Delete(ctx context.Context) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "{{.TrimmedType}}")

    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }

    return pgutils.Retry(ctx, func() error {
        return s.retryableDelete(ctx)
    })
}

func (s *storeImpl) retryableDelete(ctx context.Context) error {
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, deleteStmt); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
		}
		return errors.Wrap(err, "deleting from {{.Table}}")
	}
	return tx.Commit(ctx)
}

{{ if .GenerateDataModelHelpers -}}
// Used for Testing

// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db postgres.DB) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{.Schema.Table}} CASCADE")
}
{{- end }}
