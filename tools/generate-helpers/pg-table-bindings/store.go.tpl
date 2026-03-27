{{define "schemaVar"}}pkgSchema.{{.Table|upperCamelCase}}Schema{{end}}
{{define "commaSeparatedColumns"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "updateExclusions"}}{{range $index, $field := .}}{{if $index}}, {{end}}{{$field.ColumnName}} = EXCLUDED.{{$field.ColumnName}}{{end}}{{end}}

{{- $ := . }}
{{ $singlePK := index .Schema.PrimaryKeys 0 }}
{{ $primaryKeyName := $singlePK.ColumnName|lowerCamelCase }}
{{ $primaryKeyType := $singlePK.Type }}

package postgres

import (
    "context"
    "slices"
    "strings"
    "time"

    "github.com/hashicorp/go-multierror"
    "github.com/jackc/pgx/v5"
    "github.com/pkg/errors"
    "github.com/stackrox/rox/central/metrics"
    pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/auth/permissions"
    "github.com/stackrox/rox/pkg/logging"
    ops "github.com/stackrox/rox/pkg/metrics"
    "github.com/stackrox/rox/pkg/postgres/pgutils"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/protocompat"
    "github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
    {{- if .Jsonb }}
    "google.golang.org/protobuf/encoding/protojson"
    {{- end }}
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
    targetResource = resources.{{.ScopingResource}}
)

type (
    storeType = {{ .Type }}
    callback  = func(obj *storeType) error
)

{{- if or .NoSerialized .Jsonb }}
// Store is the interface to interact with the storage for {{ .Type }}
type Store = pgSearch.NoSerializedStore[storeType]
{{- else }}
// Store is the interface to interact with the storage for {{ .Type }}
type Store interface {
{{- if not .JoinTable }}
    Upsert(ctx context.Context, obj *storeType) error
    UpsertMany(ctx context.Context, objs []*storeType) error
    Delete(ctx context.Context, {{$primaryKeyName}} {{$primaryKeyType}}) error
    DeleteByQuery(ctx context.Context, q *v1.Query) error
    DeleteByQueryWithIDs(ctx context.Context, q *v1.Query) ([]string, error)
    DeleteMany(ctx context.Context, identifiers []{{$primaryKeyType}}) error
    PruneMany(ctx context.Context, identifiers []{{$primaryKeyType}}) error
{{- end }}

    Count(ctx context.Context, q *v1.Query) (int, error)
    Exists(ctx context.Context, {{$primaryKeyName}} {{$primaryKeyType}}) (bool, error)
    Search(ctx context.Context, q *v1.Query) ([]search.Result, error)

    Get(ctx context.Context, {{$primaryKeyName}} {{$primaryKeyType}}) (*storeType, bool, error)
{{- if .SearchCategory }}
    // Deprecated: use GetByQueryFn instead
    GetByQuery(ctx context.Context, query *v1.Query) ([]*storeType, error)
    GetByQueryFn(ctx context.Context, query *v1.Query, fn callback) error
{{- end }}
    GetMany(ctx context.Context, identifiers []{{$primaryKeyType}}) ([]*storeType, []int, error)
    GetIDs(ctx context.Context) ([]{{$primaryKeyType}}, error)

    Walk(ctx context.Context, fn callback) error
    WalkByQuery(ctx context.Context, query *v1.Query, fn callback) error

{{- if and .CachedStore .ForSAC }}
    // Deprecated: Use for SAC only
    GetAllFromCacheForSAC() []*storeType
{{- end }}
}
{{- end }}

{{ define "defineScopeChecker" }}scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_{{ . }}_ACCESS).Resource(targetResource){{ end }}

{{ define "storeCreator" -}}
    {{- if and (.CachedStore) (not .Obj.IsDirectlyScoped) -}}
        pgSearch.NewGloballyScopedGenericStoreWithCache
    {{- else if and (not .CachedStore) (not .Obj.IsDirectlyScoped) -}}
        pgSearch.NewGloballyScopedGenericStore
    {{- else if .CachedStore -}}
        pgSearch.NewGenericStoreWithCache
    {{- else -}}
        pgSearch.NewGenericStore
    {{- end -}}
{{- end }}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB) Store {
    {{- if or .NoSerialized .Jsonb }}
    return pgSearch.NewNoSerializedStore[storeType](
            db,
            schema,
            pkGetter,
            {{- if not .JoinTable }}
            {{ template "insertFunctionName" .Schema }},
            {{- if and (not .NoCopyFrom) (not .Jsonb) }}
            nil,
            {{- else }}
            nil,
            {{- end }}
            {{- else }}
            nil,
            nil,
            {{- end }}
            scanRow,
            scanRows,
            metricsSetAcquireDBConnDuration,
            metricsSetPostgresOperationDurationTime,
            {{- if .Obj.IsDirectlyScoped }}
            isUpsertAllowed,
            {{- else }}
            nil,
            {{- end }}
            targetResource,
            {{- if .NoSerialized }}
            pgSearch.NoSerializedStoreOpts[storeType]{
                BulkInsert: bulkInsertInto{{ .Schema.Table|upperCamelCase }},
            },
            {{- end }}
    )
    {{- else }}
    {{ if .CachedStore -}}
    // Use of {{ template "storeCreator" . }} can be dangerous with high cardinality stores,
    // and be the source of memory pressure. Think twice about the need for in-memory caching
    // of the whole store.
    {{ end -}}
    return {{ template "storeCreator" . }}[storeType, *storeType](
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
            {{- if .CachedStore }}
            metricsSetCacheOperationDurationTime,
            {{- end }}
            {{- if .Obj.IsDirectlyScoped }}
            isUpsertAllowed,
            {{- end }}
            targetResource,
            {{- if .DefaultSortStore }}
            pgSearch.GetDefaultSort({{.DefaultSort}}, {{.ReverseDefaultSort}}),
            {{- else }}
            nil,
            {{- end }}
            {{- if .DefaultTransform }}
            pkgSchema.{{.TransformSortOptions}},
            {{- else }}
            nil,
            {{- end }}
    )
    {{- end }}
}

// region Helper functions

func pkGetter(obj *storeType) {{$primaryKeyType}} {
    return {{ $singlePK.Getter "obj" }}
}

func metricsSetPostgresOperationDurationTime(start time.Time, op ops.Op) {
    metrics.SetPostgresOperationDurationTime(start, op, storeName)
}

func metricsSetAcquireDBConnDuration(start time.Time, op ops.Op) {
    metrics.SetAcquireDBConnDuration(start, op, storeName)
}
{{- if .CachedStore }}

func metricsSetCacheOperationDurationTime(start time.Time, op ops.Op) {
    metrics.SetCacheOperationDurationTime(start, op, storeName)
}

{{ end -}}
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
    {{- if or (eq $field.DataType "datetime") (eq $field.DataType "datetimetz") }}
        protocompat.NilOrTime({{$field.Getter "obj"}}),
    {{- else if eq $field.SQLType "uuid" }}
        pgutils.NilOrUUID({{$field.Getter "obj"}}),
    {{- else if eq $field.SQLType "cidr" }}
        pgutils.NilOrCIDR({{$field.Getter "obj"}}),
    {{- else if eq $field.DataType "map" }}
        pgutils.EmptyOrMap({{$field.Getter "obj"}}),
    {{- else if and (eq $field.DataType "string") ($field.Options.Reference) ($field.Options.Reference.Nullable) }}
        pgutils.NilOrString({{$field.Getter "obj"}}),
    {{- else }}
        {{$field.Getter "obj"}},{{end}}
{{- end}}
{{- end}}

{{- define "insertObject"}}
{{- $schema := .schema }}
func {{ template "insertFunctionName" $schema }}(batch *pgx.Batch, obj {{$schema.Type}}{{ range $field := $schema.FieldsDeterminedByParent }}, {{$field.Name}} {{$field.Type}}{{end}}) error {
    {{if and (not $schema.Parent) (not $schema.NoSerialized) }}
    {{- if $schema.Jsonb }}
    serialized, marshalErr := protojson.Marshal(obj)
    {{- else }}
    serialized, marshalErr := obj.MarshalVT()
    {{- end }}
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
    for childIndex, child := range obj.{{$child.ObjectGetter}} {
        if err := {{ template "insertFunctionName" $child }}(batch, child{{ range $field := $schema.PrimaryKeys }}, {{$field.Getter "obj"}}{{end}}, childIndex); err != nil {
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
{{- $singlePK := index $schema.PrimaryKeys 0 }}
var copyCols{{$schema.Table|upperCamelCase}} = []string{
{{- range $index, $field := $schema.DBColumnFields }}
    "{{$field.ColumnName|lowerCase}}",
{{- end }}
}

func {{ template "copyFunctionName" $schema }}(ctx context.Context, s pgSearch.Deleter, tx *postgres.Tx, {{ range $index, $field := $schema.FieldsReferringToParent }} {{$field.Name}} {{$field.Type}},{{end}} objs ...{{$schema.Type}}) error {
    if len(objs) == 0 {
        return nil
    }
    {{if not $schema.Parent }}
    {
        // CopyFrom does not upsert, so delete existing rows first to achieve upsert behavior.
        // Parent deletion cascades to children, so only the top-level parent needs deletion.
        deletes := make([]string, 0, len(objs))
        for _, obj := range objs {
            deletes = append(deletes, {{ $singlePK.Getter "obj" }})
        }
        if err := s.DeleteMany(ctx, deletes); err != nil {
            return err
        }
    }
    {{end}}

    idx := 0
    inputRows := pgx.CopyFromFunc(func() ([]any, error) {
        if idx >= len(objs) {
            return nil, nil
        }
        obj := objs[idx]
        idx++

        {{if and (not $schema.Parent) (not $schema.NoSerialized) }}
        {{- if $schema.Jsonb }}
        serialized, marshalErr := protojson.Marshal(obj)
        {{- else }}
        serialized, marshalErr := obj.MarshalVT()
        {{- end }}
        if marshalErr != nil {
            return nil, marshalErr
        }
        {{end}}

        return []interface{}{
            {{- template "insertValues" $schema }}
        }, nil
    })

    if _, err := tx.CopyFrom(ctx, pgx.Identifier{"{{$schema.Table|lowerCase}}"}, copyCols{{$schema.Table|upperCamelCase}}, inputRows); err != nil {
        return err
    }

    {{if $schema.Children }}
    for _, obj := range objs {
        {{- range $child := $schema.Children }}
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

{{- if .Jsonb }}

func scanRow(row pgx.Row) (*storeType, error) {
    var data []byte
    if err := row.Scan(&data); err != nil {
        return nil, err
    }
    msg := &storeType{}
    if err := protojson.Unmarshal(data, msg); err != nil {
        return nil, err
    }
    return msg, nil
}

func scanRows(rows pgx.Rows) (*storeType, error) {
    var data []byte
    if err := rows.Scan(&data); err != nil {
        return nil, err
    }
    msg := &storeType{}
    if err := protojson.Unmarshal(data, msg); err != nil {
        return nil, err
    }
    return msg, nil
}
{{- end }}

{{- if .NoSerialized }}

func bulkInsertInto{{ .Schema.Table|upperCamelCase }}(batch *pgx.Batch, objs []*storeType) error {
    {{- $schema := .Schema }}
    {{- $fields := $schema.DBColumnFields }}
    {{- $singlePK := index $schema.PrimaryKeys 0 }}

    // Build column arrays for unnest — one Go slice per unnestable DB column
    {{- range $field := $fields }}
    {{- if canUnnest $field }}
    arr_{{ scanVarName $field }} := make({{ unnestArrayGoType $field }}, 0, len(objs))
    {{- end }}
    {{- end }}
    {{- range $child := $schema.Children }}
    {{- range $field := $child.DBColumnFields }}
    arr_child_{{ scanVarName $field }} := make({{ unnestArrayGoType $field }}, 0, len(objs)*2)
    {{- end }}
    {{- end }}

    for _, obj := range objs {
        {{- range $field := $fields }}
        {{- if canUnnest $field }}
        arr_{{ scanVarName $field }} = append(arr_{{ scanVarName $field }}, {{ unnestAppendExpr $field }})
        {{- end }}
        {{- end }}
        {{- range $child := $schema.Children }}
        for childIdx, child := range obj.{{ getterToSetter $child.ObjectGetter }} {
            _ = child
            {{- range $field := $child.DBColumnFields }}
            {{- if eq $field.ColumnName "idx" }}
            arr_child_{{ scanVarName $field }} = append(arr_child_{{ scanVarName $field }}, childIdx)
            {{- else if $field.ObjectGetter.IsVariable }}
            arr_child_{{ scanVarName $field }} = append(arr_child_{{ scanVarName $field }}, {{ $singlePK.Getter "obj" }})
            {{- else }}
            arr_child_{{ scanVarName $field }} = append(arr_child_{{ scanVarName $field }}, child.{{ joinPath (setterPath $field) }})
            {{- end }}
            {{- end }}
        }
        {{- end }}
    }

    // Unnest INSERT for parent table (unnestable columns only)
    {{- $uFields := unnestableFields $fields }}
    batch.Queue(`INSERT INTO {{ $schema.Table }} ({{ range $index, $field := $uFields }}{{ if $index }}, {{ end }}{{ $field.ColumnName }}{{ end }})
        SELECT * FROM unnest({{ range $index, $field := $uFields }}{{ if $index }}, {{ end }}${{ add $index 1 }}::{{ pgArrayCast $field }}{{ end }})
        ON CONFLICT({{ range $index, $pk := $schema.PrimaryKeys }}{{ if $index }}, {{ end }}{{ $pk.ColumnName }}{{ end }}) DO UPDATE SET
        {{ range $index, $field := $uFields }}{{ if $index }}, {{ end }}{{ $field.ColumnName }} = EXCLUDED.{{ $field.ColumnName }}{{ end }}`,
        {{- range $field := $uFields }}
        arr_{{ scanVarName $field }},
        {{- end }}
    )

    // Per-row UPDATE for columns that can't be unnested (arrays, maps)
    {{- range $field := nonUnnestableFields $fields }}
    for _, obj := range objs {
        batch.Queue(`UPDATE {{ $schema.Table }} SET {{ $field.ColumnName }} = $1 WHERE {{ $singlePK.ColumnName }} = $2`,
            {{ unnestAppendExpr $field }}, {{ $singlePK.Getter "obj" }},
        )
    }
    {{- end }}

    {{- range $child := $schema.Children }}
    // Delete existing children, then bulk insert
    batch.Queue(`DELETE FROM {{ $child.Table }} WHERE {{ (index $child.FieldsReferringToParent 0).ColumnName }} = ANY($1::{{ pgArrayCast $singlePK }})`,
        arr_{{ scanVarName $singlePK }},
    )
    {{- $firstChildField := index $child.DBColumnFields 0 }}
    if len(arr_child_{{ scanVarName $firstChildField }}) > 0 {
        batch.Queue(`INSERT INTO {{ $child.Table }} ({{ range $index, $field := $child.DBColumnFields }}{{ if $index }}, {{ end }}{{ $field.ColumnName }}{{ end }})
            SELECT * FROM unnest({{ range $index, $field := $child.DBColumnFields }}{{ if $index }}, {{ end }}${{ add $index 1 }}::{{ pgArrayCast $field }}{{ end }})`,
            {{- range $field := $child.DBColumnFields }}
            arr_child_{{ scanVarName $field }},
            {{- end }}
        )
    }
    {{- end }}

    return nil
}

func buildFromScan(
    {{- range $index, $field := .Schema.DBColumnFields }}
    {{ scanVarName $field }} {{ scanVarType $field }},
    {{- end }}
) *storeType {
    obj := &storeType{}
    {{- range $sub := .Schema.InlinedSubMessages }}
    obj.{{ $sub.FieldName }} = &{{ $sub.TypeName }}{}
    {{- end }}
    {{- range $field := .Schema.DBColumnFields }}
    {{- $setter := fieldSetterExpr $field }}
    {{- if $setter }}
    {{- if needsTypeConversion $field }}
    {{ $setter }} = {{ typeConversionExpr $field (scanVarName $field) }}
    {{- else }}
    {{ $setter }} = {{ scanVarName $field }}
    {{- end }}
    {{- end }}
    {{- end }}
    return obj
}

func scanRow(row pgx.Row) (*storeType, error) {
    {{- range $field := .Schema.DBColumnFields }}
    var {{ scanVarName $field }} {{ scanVarType $field }}
    {{- end }}

    if err := row.Scan(
        {{- range $field := .Schema.DBColumnFields }}
        &{{ scanVarName $field }},
        {{- end }}
    ); err != nil {
        return nil, err
    }

    return buildFromScan(
        {{- range $field := .Schema.DBColumnFields }}
        {{ scanVarName $field }},
        {{- end }}
    ), nil
}

{{- if .Schema.Children }}
// FetchChildren populates child table data (repeated message fields) for the given objects.
// This is NOT called automatically by the store's read methods — callers must opt in.
func FetchChildren(ctx context.Context, db postgres.DB, objs []*storeType) error {
    if len(objs) == 0 {
        return nil
    }
    objsByID := make(map[string]*storeType, len(objs))
    ids := make([]string, 0, len(objs))
    for _, obj := range objs {
        id := pkGetter(obj)
        objsByID[id] = obj
        ids = append(ids, id)
    }
    {{- range $child := .Schema.Children }}
    {{- $parentPKs := $.Schema.PrimaryKeys }}
    {
        q := "SELECT {{ range $index, $field := $child.DBColumnFields }}{{if $index}}, {{end}}{{$field.ColumnName}}{{end}} FROM {{$child.Table}} WHERE {{range $index, $field := $child.FieldsReferringToParent}}{{if $index}} AND {{end}}{{$field.ColumnName}} = ANY(${{add $index 1}}::uuid[]){{end}} ORDER BY {{range $index, $field := $child.FieldsReferringToParent}}{{if $index}}, {{end}}{{$field.ColumnName}}{{end}}, idx"
        rows, err := db.Query(ctx, q, ids)
        if err != nil {
            return err
        }
        defer rows.Close()
        for rows.Next() {
            {{- range $field := $child.DBColumnFields }}
            var {{ scanVarName $field }} {{ scanVarType $field }}
            {{- end }}
            if err := rows.Scan(
                {{- range $field := $child.DBColumnFields }}
                &{{ scanVarName $field }},
                {{- end }}
            ); err != nil {
                return err
            }
            {{- $parentIDField := index $child.FieldsReferringToParent 0 }}
            parent, ok := objsByID[{{ scanVarName $parentIDField }}]
            if !ok {
                continue
            }
            {{- range $sub := $.Schema.InlinedSubMessages }}
            if parent.{{ $sub.FieldName }} == nil {
                parent.{{ $sub.FieldName }} = &{{ $sub.TypeName }}{}
            }
            {{- end }}
            child := &{{ stripPointer $child.Type }}{}
            {{- range $field := $child.DBColumnFields }}
            {{- if not $field.ObjectGetter.IsVariable }}
            {{- $setter := fieldSetterExpr $field }}
            {{- if $setter }}
            {{- $childSetter := (printf "child.%s" (joinPath (setterPath $field))) }}
            {{- if needsTypeConversion $field }}
            {{ $childSetter }} = {{ typeConversionExpr $field (scanVarName $field) }}
            {{- else }}
            {{ $childSetter }} = {{ scanVarName $field }}
            {{- end }}
            {{- end }}
            {{- end }}
            {{- end }}
            parent.{{ getterToSetter $child.ObjectGetter }} = append(parent.{{ getterToSetter $child.ObjectGetter }}, child)
        }
        if err := rows.Err(); err != nil {
            return err
        }
    }
    {{- end }}
    return nil
}
{{- end }}

func scanRows(rows pgx.Rows) (*storeType, error) {
    {{- range $field := .Schema.DBColumnFields }}
    var {{ scanVarName $field }} {{ scanVarType $field }}
    {{- end }}

    if err := rows.Scan(
        {{- range $field := .Schema.DBColumnFields }}
        &{{ scanVarName $field }},
        {{- end }}
    ); err != nil {
        return nil, err
    }

    return buildFromScan(
        {{- range $field := .Schema.DBColumnFields }}
        {{ scanVarName $field }},
        {{- end }}
    ), nil
}
{{- end }}

// endregion Helper functions

{{- define "dropTableFunctionName"}}dropTable{{.Table | upperCamelCase}}{{end}}

{{- define "dropTable"}}
{{- $schema := . }}
func {{ template "dropTableFunctionName" $schema }}(ctx context.Context, db postgres.DB) {
    _, _ = db.Exec(ctx, "DROP TABLE IF EXISTS {{$schema.Table}} CASCADE")
    {{range $child := $schema.Children}}{{ template "dropTableFunctionName" $child }}(ctx, db)
    {{end}}
}
{{range $child := $schema.Children}}{{ template "dropTable" $child }}{{end}}
{{- end}}

{{ if .GenerateDataModelHelpers -}}
// region Used for testing

// CreateTableAndNewStore returns a new Store instance for testing.
func CreateTableAndNewStore(ctx context.Context, db postgres.DB, gormDB *gorm.DB) Store {
	pkgSchema.ApplySchemaForTable(ctx, gormDB, baseTable)
	return New(db)
}

// Destroy drops the tables associated with the target object type.
func Destroy(ctx context.Context, db postgres.DB) {
    {{template "dropTableFunctionName" .Schema}}(ctx, db)
}

{{template "dropTable" .Schema}}

// endregion Used for testing
{{- end }}
