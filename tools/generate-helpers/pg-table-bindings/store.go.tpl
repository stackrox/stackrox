{{define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{define "paramList"}}{{range $index, $pk := .}}{{if $index}}, {{end}}{{$pk.ColumnName|lowerCamelCase}} {{$pk.Type}}{{end}}{{end}}
{{define "argList"}}{{range $index, $pk := .}}{{if $index}}, {{end}}{{$pk.ColumnName|lowerCamelCase}}{{end}}{{end}}
{{define "whereMatch"}}{{range $index, $pk := .}}{{if $index}} AND {{end}}{{$pk.ColumnName}} = ${{add $index 1}}{{end}}{{end}}
{{define "commaSeparatedColumns"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}
{{define "matchQuery" -}}
    {{- $pks := index . 0 -}}
    {{- $singlePK := index . 1 -}}
    {{- range $index, $pk := $pks -}}
    {{- if eq $pk.Name $singlePK.Name -}}
        search.NewQueryBuilder().AddDocIDs({{ $singlePK.ColumnName|lowerCamelCase }}).ProtoQuery(),
    {{- else }}
        search.NewQueryBuilder().AddExactMatches(search.FieldLabel("{{ searchFieldNameInOtherSchema $pk }}"), {{ $pk.ColumnName|lowerCamelCase }}).ProtoQuery(),
    {{- end -}}
    {{- end -}}
{{end}}

{{- $ := . }}
{{- $pks := .Schema.PrimaryKeys }}

{{- $singlePK := false }}
{{- if eq (len $pks) 1 }}
{{ $singlePK = index $pks 0 }}
{{/*If there are multiple pks, then use the explicitly specified ID column.*/}}
{{- else if .Schema.ID.ColumnName}}
{{ $singlePK = .Schema.ID }}
{{- end }}

package postgres

import (
    "context"
    "strings"
    "time"

    "github.com/hashicorp/go-multierror"
    "github.com/jackc/pgx/v4"
    "github.com/pkg/errors"
    "github.com/stackrox/rox/central/metrics"
    "github.com/stackrox/rox/central/role/resources"
    pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/auth/permissions"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/sac"
    "github.com/stackrox/rox/pkg/search"
    pgSearch "github.com/stackrox/rox/pkg/search/postgres"
    "github.com/stackrox/rox/pkg/sync"
    "github.com/stackrox/rox/pkg/utils"
    "github.com/stackrox/rox/pkg/uuid"
    "gorm.io/gorm"
)

const (
        baseTable = {{ .Table | quote }}
        storeName = {{ .TrimmedType | quote }}

        batchAfter = 100

        // using copyFrom, we may not even want to batch.  It would probably be simpler
        // to deal with failures if we just sent it all.  Something to think about as we
        // proceed and move into more e2e and larger performance testing
        batchSize = 10000
)

var (
    log = logging.LoggerForModule()
    schema = {{ template "schemaVar" .Schema}}
    {{- if or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
        targetResource = resources.{{.Type | storageToResource}}
    {{- end }}
)

// Store is the interface to interact with the storage for {{.Type}}
type Store interface {
{{- if not .JoinTable }}
    Upsert(ctx context.Context, obj *{{.Type}}) error
    UpsertMany(ctx context.Context, objs []*{{.Type}}) error
    Delete(ctx context.Context, {{template "paramList" $pks}}) error
    DeleteByQuery(ctx context.Context, q *v1.Query) error
{{- if $singlePK }}
    DeleteMany(ctx context.Context, identifiers []{{$singlePK.Type}}) error
{{- end }}
{{- end }}

    Count(ctx context.Context) (int, error)
    Exists(ctx context.Context, {{template "paramList" $pks}}) (bool, error)

    Get(ctx context.Context, {{template "paramList" $pks}}) (*{{.Type}}, bool, error)
{{- if .SearchCategory }}
    GetByQuery(ctx context.Context, query *v1.Query) ([]*{{.Type}}, error)
{{- end }}
{{- if $singlePK }}
    GetMany(ctx context.Context, identifiers []{{$singlePK.Type}}) ([]*{{.Type}}, []int, error)
    GetIDs(ctx context.Context) ([]{{$singlePK.Type}}, error)
{{- end }}
{{- if .GetAll }}
    GetAll(ctx context.Context) ([]*{{.Type}}, error)
{{- end }}

    Walk(ctx context.Context, fn func(obj *{{.Type}}) error) error
}

type storeImpl struct {
    *pgSearch.GenericStore[{{.Type}}, *{{.Type}}]
    db postgres.DB
    mutex sync.RWMutex
}

{{ define "defineScopeChecker" }}scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_{{ . }}_ACCESS).Resource(targetResource){{ end }}

{{define "createTableStmtVar"}}pkgSchema.CreateTable{{.Table|upperCamelCase}}Stmt{{end}}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
    return &storeImpl{
        db: db,
        GenericStore: pgSearch.NewGenericStore{{ if .PermissionChecker }}WithPermissionChecker{{ end }}[{{.Type}}, *{{.Type}}](
            db,
            schema,
            pkGetter,
            metricsSetAcquireDBConnDuration,
            metricsSetPostgresOperationDurationTime,
            {{ if .PermissionChecker }}{{ .PermissionChecker }}{{ else }}targetResource{{ end }},
        ),
    }
}

// region Helper functions

func pkGetter(obj *{{ .Type }}) {{$singlePK.Type}} {
    return {{ $singlePK.Getter "obj" }}
}

func metricsSetPostgresOperationDurationTime(start time.Time, op ops.Op) {
    metrics.SetPostgresOperationDurationTime(start, op, storeName)
}

func metricsSetAcquireDBConnDuration(start time.Time, op ops.Op) {
    metrics.SetAcquireDBConnDuration(start, op, storeName)
}

{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := .schema }}
func {{ template "insertFunctionName" $schema }}({{ if eq (len $schema.Children) 0 }}_{{ else }}ctx{{ end }} context.Context, batch *pgx.Batch, obj {{$schema.Type}}{{ range $field := $schema.FieldsDeterminedByParent }}, {{$field.Name}} {{$field.Type}}{{end}}) error {
    {{if not $schema.Parent }}
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }
    {{end}}

    values := []interface{} {
        // parent primary keys start
        {{- range $field := $schema.DBColumnFields -}}
        {{- if eq $field.DataType "datetime" }}
        pgutils.NilOrTime({{$field.Getter "obj"}}),
        {{- else if eq $field.SQLType "uuid" }}
        pgutils.NilOrUUID({{$field.Getter "obj"}}),
        {{- else }}
        {{$field.Getter "obj"}},{{end}}
        {{- end}}
    }

    finalStr := "INSERT INTO {{$schema.Table}} ({{template "commaSeparatedColumns" $schema.DBColumnFields }}) VALUES({{ valueExpansion (len $schema.DBColumnFields) }}) ON CONFLICT({{template "commaSeparatedColumns" $schema.PrimaryKeys}}) DO UPDATE SET {{template "updateExclusions" $schema.DBColumnFields}}"
    batch.Queue(finalStr, values...)

    {{ if $schema.Children }}
    var query string
    {{end}}

    {{range $index, $child := $schema.Children }}
    for childIndex, child := range obj.{{$child.ObjectGetter}} {
        if err := {{ template "insertFunctionName" $child }}(ctx, batch, child{{ range $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, childIndex); err != nil {
            return err
        }
    }

    query = "delete from {{$child.Table}} where {{ range $index, $field := $child.FieldsReferringToParent }}{{if $index}} AND {{end}}{{$field.ColumnName}} = ${{add $index 1}}{{end}} AND idx >= ${{add (len $child.FieldsReferringToParent) 1}}"
    batch.Queue(query{{ range $field := $schema.PrimaryKeys }}, {{if eq $field.SQLType "uuid"}}pgutils.NilOrUUID({{end}}{{$field.Getter "obj"}}{{if eq $field.SQLType "uuid"}}){{end}}{{end}}, len(obj.{{$child.ObjectGetter}}))
    {{- end}}
    return nil
}

{{range $index, $child := $schema.Children}}{{ template "insertObject" dict "schema" $child "joinTable" false }}{{end}}
{{- end}}

{{- if not .JoinTable }}
{{ template "insertObject" dict "schema" .Schema "joinTable" .JoinTable }}
{{- end}}

{{- define "copyFunctionName"}}{{- $schema := . }}copyFrom{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "copyObject"}}
{{- $schema := .schema }}
func (s *storeImpl) {{ template "copyFunctionName" $schema }}(ctx context.Context, tx *postgres.Tx, {{ range $index, $field := $schema.FieldsReferringToParent }} {{$field.Name}} {{$field.Type}},{{end}} objs ...{{$schema.Type}}) error {

    inputRows := [][]interface{}{}

    var err error

    {{if and (eq (len $schema.PrimaryKeys) 1) (not $schema.Parent) }}
    // This is a copy so first we must delete the rows and re-add them
    // Which is essentially the desired behaviour of an upsert.
    var deletes []string
    {{end}}

    copyCols := []string {
    {{range $index, $field := $schema.DBColumnFields}}
        "{{$field.ColumnName|lowerCase}}",
    {{end}}
    }

    for idx, obj := range objs {
        // Todo: ROX-9499 Figure out how to more cleanly template around this issue.
        log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj "+
		"in the loop is not used as it only consists of the parent ID and the index.  Putting this here as a stop gap "+
		"to simply use the object.  %s", obj)

        {{/* If embedded, the top-level has the full serialized object */}}
        {{if not $schema.Parent }}
        serialized, marshalErr := obj.Marshal()
        if marshalErr != nil {
            return marshalErr
        }
        {{end}}

        inputRows = append(inputRows, []interface{}{
            {{ range $index, $field := $schema.DBColumnFields }}
            {{if eq $field.DataType "datetime"}}
            pgutils.NilOrTime({{$field.Getter "obj"}}),
            {{- else if eq $field.SQLType "uuid" }}
            pgutils.NilOrUUID({{$field.Getter "obj"}}),
            {{- else}}
            {{$field.Getter "obj"}},{{end}}
            {{end}}
        })

        {{ if not $schema.Parent }}
        {{if eq (len $schema.PrimaryKeys) 1}}
        // Add the ID to be deleted.
        deletes = append(deletes, {{ range $field := $schema.PrimaryKeys }}{{$field.Getter "obj"}}, {{end}})
        {{else}}
        if err := s.Delete(ctx, {{ range $field := $schema.PrimaryKeys }}{{$field.Getter "obj"}}, {{end}}); err != nil {
            return err
        }

        {{end}}
        {{end}}

        // if we hit our batch size we need to push the data
        if (idx + 1) % batchSize == 0 || idx == len(objs) - 1  {
            // copy does not upsert so have to delete first.  parent deletion cascades so only need to
            // delete for the top level parent
            {{if and ((eq (len $schema.PrimaryKeys) 1)) (not $schema.Parent) }}
            if err := s.DeleteMany(ctx, deletes); err != nil {
                return err
            }
            // clear the inserts and vals for the next batch
            deletes = nil
            {{end}}

            _, err = tx.CopyFrom(ctx, pgx.Identifier{"{{$schema.Table|lowerCase}}"}, copyCols, pgx.CopyFromRows(inputRows))

            if err != nil {
                return err
            }

            // clear the input rows for the next batch
            inputRows = inputRows[:0]
        }
    }

    {{if $schema.Children }}
    for idx, obj := range objs {
        _ = idx // idx may or may not be used depending on how nested we are, so avoid compile-time errors.
        {{range $child := $schema.Children }}
        if err = s.{{ template "copyFunctionName" $child }}(ctx, tx{{ range $index, $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, obj.{{$child.ObjectGetter}}...); err != nil {
            return err
        }
        {{- end}}
    }
    {{end}}

    return err
}
{{range $child := $schema.Children}}{{ template "copyObject" dict "schema" $child }}{{end}}
{{- end}}

{{- if not .JoinTable }}
{{- if not .NoCopyFrom }}
{{ template "copyObject" dict "schema" .Schema }}
{{- end }}
{{- end }}

{{- if not .JoinTable }}
{{- if not .NoCopyFrom }}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*{{.Type}}) error {
    conn, err := s.AcquireConn(ctx, ops.Get)
	if err != nil {
	    return err
	}
    defer conn.Release()

    tx, err := conn.Begin(ctx)
    if err != nil {
        return err
    }

    if err := s.{{ template "copyFunctionName" .Schema }}(ctx, tx, objs...); err != nil {
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
{{- end}}

func (s *storeImpl) upsert(ctx context.Context, objs ...*{{.Type}}) error {
    conn, err := s.AcquireConn(ctx, ops.Get)
	if err != nil {
	    return err
	}
	defer conn.Release()

    for _, obj := range objs {
        batch := &pgx.Batch{}
	    if err := {{ template "insertFunctionName" .Schema }}(ctx, batch, obj); err != nil {
		    return err
        }
		batchResults := conn.SendBatch(ctx, batch)
		var result *multierror.Error
		for i := 0; i < batch.Len(); i++ {
			_, err := batchResults.Exec()
			result = multierror.Append(result, err)
		}
		if err := batchResults.Close(); err != nil {
			return err
		}
		if err := result.ErrorOrNil(); err != nil {
			return err
		}
    }
    return nil
}
{{- end }}

// endregion Helper functions

//// Interface functions

{{- if not .JoinTable }}

// Upsert saves the current state of an object in storage.
func (s *storeImpl) Upsert(ctx context.Context, obj *{{.Type}}) error {
    defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "{{.TrimmedType}}")

    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.UpsertAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsGloballyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    {{- else if and (.Obj.IsDirectlyScoped) (.Obj.IsClusterScope) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}.
        ClusterID({{ "obj" | .Obj.GetClusterID }})
    {{- else if and (.Obj.IsDirectlyScoped) (.Obj.IsNamespaceScope) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}.
        ClusterID({{ "obj" | .Obj.GetClusterID }}).Namespace({{ "obj" | .Obj.GetNamespace }})
    {{- end }}
    {{- if or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped)  }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- end }}

	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}
{{- end }}

{{- if not .JoinTable }}

// UpsertMany saves the state of multiple objects in the storage.
func (s *storeImpl) UpsertMany(ctx context.Context, objs []*{{.Type}}) error {
    defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "{{.TrimmedType}}")

    {{ if .PermissionChecker -}}
    if ok, err := {{ .PermissionChecker }}.UpsertManyAllowed(ctx); err != nil {
        return err
    } else if !ok {
        return sac.ErrResourceAccessDenied
    }
    {{- else if or (.Obj.IsGloballyScoped) (.Obj.IsIndirectlyScoped) }}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        return sac.ErrResourceAccessDenied
    }
    {{- else if .Obj.IsDirectlyScoped -}}
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if !scopeChecker.IsAllowed() {
        var deniedIDs []string
        for _, obj := range objs {
            {{- if .Obj.IsClusterScope }}
            subScopeChecker := scopeChecker.ClusterID({{ "obj" | .Obj.GetClusterID }})
            {{- else if .Obj.IsNamespaceScope }}
            subScopeChecker := scopeChecker.ClusterID({{ "obj" | .Obj.GetClusterID }}).Namespace({{ "obj" | .Obj.GetNamespace }})
            {{- end }}
            if !subScopeChecker.IsAllowed() {
                deniedIDs = append(deniedIDs, {{ "obj" | .Obj.GetID }})
            }
        }
        if len(deniedIDs) != 0 {
            return errors.Wrapf(sac.ErrResourceAccessDenied, "modifying {{ .TrimmedType|lowerCamelCase }}s with IDs [%s] was denied", strings.Join(deniedIDs, ", "))
        }
    }
    {{- end }}

    {{- if .NoCopyFrom }}
    return s.upsert(ctx, objs...)
    {{- else }}

	return pgutils.Retry(func() error {
		// Lock since copyFrom requires a delete first before being executed.  If multiple processes are updating
		// same subset of rows, both deletes could occur before the copyFrom resulting in unique constraint
		// violations
		if len(objs) < batchAfter {
		    s.mutex.RLock()
		    defer s.mutex.RUnlock()

		    return s.upsert(ctx, objs...)
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()

		return s.copyFrom(ctx, objs...)
	})
    {{- end }}
}
{{- end }}

//// Interface functions - END

//// Used for testing

// CreateTableAndNewStore returns a new Store instance for testing.
func CreateTableAndNewStore(ctx context.Context, db postgres.DB, gormDB *gorm.DB) Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return New(db)
}

{{- define "dropTableFunctionName"}}dropTable{{.Table | upperCamelCase}}{{end}}


// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db postgres.DB) {
    {{template "dropTableFunctionName" .Schema}}(ctx, db)
}

{{- define "dropTable"}}
{{- $schema := . }}
func {{ template "dropTableFunctionName" $schema }}(ctx context.Context, db postgres.DB) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{$schema.Table}} CASCADE")
    {{range $child := $schema.Children}}{{ template "dropTableFunctionName" $child }}(ctx, db)
    {{end}}
}
{{range $child := $schema.Children}}{{ template "dropTable" $child }}{{end}}
{{- end}}

{{template "dropTable" .Schema}}

//// Used for testing - END
