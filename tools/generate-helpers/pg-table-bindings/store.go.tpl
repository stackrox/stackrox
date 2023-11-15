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

{{ $singlePK := index $pks 0 }}

package postgres

import (
    "context"
    "strings"
    "time"

    "github.com/hashicorp/go-multierror"
    "github.com/jackc/pgx/v5"
    "github.com/pkg/errors"
    "github.com/stackrox/rox/central/metrics"
    "github.com/stackrox/rox/pkg/features"
    pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/auth/permissions"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
    "github.com/stackrox/rox/pkg/search"
    pgSearch "github.com/stackrox/rox/pkg/search/postgres"
    "github.com/stackrox/rox/pkg/utils"
    "github.com/stackrox/rox/pkg/uuid"
    "gorm.io/gorm"
)

const (
        baseTable = {{ .Table | quote }}
        storeName = {{ .TrimmedType | quote }}
)

var (
    log = logging.LoggerForModule()
    schema = {{ template "schemaVar" .Schema}}
    {{- if or (.Obj.IsGloballyScoped) (.Obj.IsDirectlyScoped) (.Obj.IsIndirectlyScoped) }}
        targetResource = resources.{{.Type | storageToResource}}
    {{- end }}
)

type storeType = {{ .Type }}

// Store is the interface to interact with the storage for {{ .Type }}
type Store interface {
{{- if not .JoinTable }}
    Upsert(ctx context.Context, obj *storeType) error
    UpsertMany(ctx context.Context, objs []*storeType) error
    Delete(ctx context.Context, {{template "paramList" $pks}}) error
    DeleteByQuery(ctx context.Context, q *v1.Query) error
    DeleteMany(ctx context.Context, identifiers []{{$singlePK.Type}}) error
{{- end }}

    Count(ctx context.Context) (int, error)
    Exists(ctx context.Context, {{template "paramList" $pks}}) (bool, error)

    Get(ctx context.Context, {{template "paramList" $pks}}) (*storeType, bool, error)
{{- if .SearchCategory }}
    GetByQuery(ctx context.Context, query *v1.Query) ([]*storeType, error)
{{- end }}
    GetMany(ctx context.Context, identifiers []{{$singlePK.Type}}) ([]*storeType, []int, error)
    GetIDs(ctx context.Context) ([]{{$singlePK.Type}}, error)
{{- if .GetAll }}
    GetAll(ctx context.Context) ([]*storeType, error)
{{- end }}

    Walk(ctx context.Context, fn func(obj *storeType) error) error
}

{{ define "defineScopeChecker" }}scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_{{ . }}_ACCESS).Resource(targetResource){{ end }}

{{define "createTableStmtVar"}}pkgSchema.CreateTable{{.Table|upperCamelCase}}Stmt{{end}}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
    return pgSearch.NewGenericStore{{ if .PermissionChecker }}WithPermissionChecker{{ end }}[storeType, *storeType](
            db,
            schema,
            pkGetter,
            {{- if not .JoinTable }}
            {{ template "insertFunctionName" .Schema }},
                {{- if not .NoCopyFrom }}
                {{ template "copyFunctionName" .Schema }},
                {{- else }}
                nil,
                {{- end }}
            {{- else }}
            nil,
            nil,
            {{- end }}
            metricsSetAcquireDBConnDuration,
            metricsSetPostgresOperationDurationTime,
            {{- if or (.Obj.IsGloballyScoped) (.Obj.IsIndirectlyScoped) }}
            pgSearch.GloballyScopedUpsertChecker[storeType, *storeType](targetResource),
            {{- else if .Obj.IsDirectlyScoped }}
            isUpsertAllowed,
            {{- end }}
            {{ if .PermissionChecker }}{{ .PermissionChecker }}{{ else }}targetResource{{ end }},
    )
}

// region Helper functions

func pkGetter(obj *storeType) {{$singlePK.Type}} {
    return {{ $singlePK.Getter "obj" }}
}

func metricsSetPostgresOperationDurationTime(start time.Time, op ops.Op) {
    metrics.SetPostgresOperationDurationTime(start, op, storeName)
}

func metricsSetAcquireDBConnDuration(start time.Time, op ops.Op) {
    metrics.SetAcquireDBConnDuration(start, op, storeName)
}

{{- if .Obj.IsDirectlyScoped }}
func isUpsertAllowed(ctx context.Context, objs ...*storeType) error {
    {{ template "defineScopeChecker" "READ_WRITE" }}
    if scopeChecker.IsAllowed() {
        return nil
    }
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
    return nil
}
{{- end }}


{{- define "insertFunctionName"}}{{- $schema := . }}insertInto{{$schema.Table|upperCamelCase}}
{{- end}}

{{- define "insertValues"}}{{- $schema := . -}}
{{- range $field := $schema.DBColumnFields -}}
    {{- if eq $field.DataType "datetime" }}
        pgutils.NilOrTime({{$field.Getter "obj"}}),
    {{- else if eq $field.SQLType "uuid" }}
        pgutils.NilOrUUID({{$field.Getter "obj"}}),
    {{- else if eq $field.DataType "map" }}
        pgutils.EmptyOrMap({{$field.Getter "obj"}}),
    {{- else }}
        {{$field.Getter "obj"}},{{end}}
{{- end}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := .schema }}
func {{ template "insertFunctionName" $schema }}(batch *pgx.Batch, obj {{$schema.Type}}{{ range $field := $schema.FieldsDeterminedByParent }}, {{$field.Name}} {{$field.Type}}{{end}}) error {
    {{if not $schema.Parent }}
    serialized, marshalErr := obj.Marshal()
    if marshalErr != nil {
        return marshalErr
    }
    {{end}}

    values := []interface{} {
        // parent primary keys start
        {{- template "insertValues" $schema }}
    }

    finalStr := "INSERT INTO {{$schema.Table}} ({{template "commaSeparatedColumns" $schema.DBColumnFields }}) VALUES({{ valueExpansion (len $schema.DBColumnFields) }}) ON CONFLICT({{template "commaSeparatedColumns" $schema.PrimaryKeys}}) DO UPDATE SET {{template "updateExclusions" $schema.DBColumnFields}}"
    batch.Queue(finalStr, values...)

    {{ if $schema.Children }}
    var query string
    {{end}}

    {{range $index, $child := $schema.Children }}
    {{ if $child.Flag }}
    if features.Flags["{{$child.Flag}}"].Enabled() {
    {{- end }}
    for childIndex, child := range obj.{{$child.ObjectGetter}} {
        if err := {{ template "insertFunctionName" $child }}(batch, child{{ range $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, childIndex); err != nil {
            return err
        }
    }

    query = "delete from {{$child.Table}} where {{ range $index, $field := $child.FieldsReferringToParent }}{{if $index}} AND {{end}}{{$field.ColumnName}} = ${{add $index 1}}{{end}} AND idx >= ${{add (len $child.FieldsReferringToParent) 1}}"
    batch.Queue(query{{ range $field := $schema.PrimaryKeys }}, {{if eq $field.SQLType "uuid"}}pgutils.NilOrUUID({{end}}{{$field.Getter "obj"}}{{if eq $field.SQLType "uuid"}}){{end}}{{end}}, len(obj.{{$child.ObjectGetter}}))
    {{- if $child.Flag }}
    }
    {{- end}}
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
func {{ template "copyFunctionName" $schema }}(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, {{ range $index, $field := $schema.FieldsReferringToParent }} {{$field.Name}} {{$field.Type}},{{end}} objs ...{{$schema.Type}}) error {
    batchSize := pgSearch.MaxBatchSize
    if len(objs) < batchSize {
        batchSize = len(objs)
    }
    inputRows := make([][]interface{}, 0, batchSize)
    {{if not $schema.Parent }}
    // This is a copy so first we must delete the rows and re-add them
    // Which is essentially the desired behaviour of an upsert.
    deletes := make([]string, 0, batchSize)
    {{end}}

    copyCols := []string {
    {{- range $index, $field := $schema.DBColumnFields }}
        "{{$field.ColumnName|lowerCase}}",
    {{- end }}
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
            {{- template "insertValues" $schema }}
        })

        {{ if not $schema.Parent }}
        // Add the ID to be deleted.
        deletes = append(deletes, {{ range $field := $schema.PrimaryKeys }}{{$field.Getter "obj"}}, {{end}})
        {{end}}

        // if we hit our batch size we need to push the data
        if (idx + 1) % batchSize == 0 || idx == len(objs) - 1  {
            // copy does not upsert so have to delete first.  parent deletion cascades so only need to
            // delete for the top level parent
            {{if not $schema.Parent }}
            if err := s.DeleteMany(ctx, deletes); err != nil {
                return err
            }
            // clear the inserts and vals for the next batch
            deletes = deletes[:0]
            {{end}}
            if _, err := tx.CopyFrom(ctx, pgx.Identifier{"{{$schema.Table|lowerCase}}"}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
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
        if err := {{ template "copyFunctionName" $child }}(ctx, s, tx{{ range $index, $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, obj.{{$child.ObjectGetter}}...); err != nil {
            return err
        }
        {{- end}}
    }
    {{end}}
    return nil
}
{{range $child := $schema.Children}}{{ template "copyObject" dict "schema" $child }}{{end}}
{{- end}}

{{- if not .JoinTable }}
{{- if not .NoCopyFrom }}
{{ template "copyObject" dict "schema" .Schema }}
{{- end }}
{{- end }}

// endregion Helper functions

// region Used for testing

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

// endregion Used for testing
