{{define "schemaVar"}}{{.|upperCamelCase}}Schema{{end}}
{{define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}

package schema

import (
    "context"
    "fmt"
    "reflect"
    "time"

    "github.com/lib/pq"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/postgres/walker"
    "github.com/stackrox/rox/pkg/search"
)

{{- define "createTableStmt" }}
{{- $schema := . }}
&postgres.CreateStmts{
    GormModel: (*{{$schema.Table|upperCamelCase}})(nil),
    Children: []*postgres.CreateStmts{
     {{- range $idx, $child := $schema.Children }}
        {{- template "createTableStmt" $child }},
    {{- end }}
    },
}
{{- end}}

var (
    // {{template "createTableStmtVar" .Schema }} holds the create statement for table `{{.Schema.Table|lowerCase}}`.
    {{template "createTableStmtVar" .Schema }} = {{template "createTableStmt" .Schema }}

    // {{template "schemaVar" .Schema.Table}} is the go schema for table `{{.Schema.Table|lowerCase}}`.
    {{template "schemaVar" .Schema.Table}} = func() *walker.Schema {
        {{- if .RegisterSchema }}
        schema := GetSchemaForTable("{{.Schema.Table}}")
        if schema != nil {
            return schema
        }
        schema = walker.Walk(reflect.TypeOf(({{.Schema.Type}})(nil)), "{{.Schema.Table}}")
        {{- else}}
        schema := walker.Walk(reflect.TypeOf(({{.Schema.Type}})(nil)), "{{.Schema.Table}}")
        {{- end}}

        {{- if gt (len .References) 0 }}
		referencedSchemas := map[string]*walker.Schema{
		{{- range $ref := .References }}
		    "{{ $ref.TypeName }}": {{ template "schemaVar" $ref.Table }},
		{{- end }}
		}

        schema.ResolveReferences(func(messageTypeName string) *walker.Schema {
             return referencedSchemas[fmt.Sprintf("storage.%s", messageTypeName)]
         })
         {{- end }}

        {{- if .SearchCategory }}
            schema.SetOptionsMap(search.Walk(v1.{{.SearchCategory}}, "{{.Schema.TypeName|lower}}", ({{.Schema.Type}})(nil)))
            {{- if .SearchScope }}
            schema.SetSearchScope([]v1.SearchCategory{
            {{- range $category := .SearchScope }}
                {{$category}},
            {{- end}}
            }...)
            {{- end }}
        {{- end }}
        {{- if .RegisterSchema }}
        RegisterTable(schema, {{template "createTableStmtVar" .Schema }})
        {{- end}}
        return schema
    }()
)

{{- define "createGormModel" }}
{{- $schema := . }}
    // {{$schema.Table|upperCamelCase}} holds the Gorm model for Postgres table `{{$schema.Table|lowerCase}}`.
    type {{$schema.Table|upperCamelCase}} struct {
    {{- range $idx, $field := $schema.DBColumnFields }}
        {{$field.ColumnName|upperCamelCase}} {{$field.ModelType}} `gorm:"{{- /**/ -}}
        column:{{$field.ColumnName|lowerCase}};{{- /**/ -}}
        type:{{$field.SQLType}}{{if $field.Options.Unique}};unique{{end}}{{if $field.Options.PrimaryKey}};primaryKey{{end}}{{- /**/ -}}
        {{if $field.Options.Index}};{{- /**/ -}}
            index:{{$schema.Table|lowerCamelCase|lowerCase}}_{{$field.ColumnName|lowerCase}},{{- /**/ -}}
            type:{{$field.Options.Index}}{{- /**/ -}}
        {{end}}{{- /**/ -}}
        "`
    {{- end}}
    {{- range $idx, $rel := $schema.RelationshipsToDefineAsForeignKeys }}
        {{$rel.OtherSchema.Table|upperCamelCase}}{{if $rel.CycleReference}}Cycle{{end}}Ref {{$rel.OtherSchema.Table|upperCamelCase}} `gorm:"{{- /**/ -}}
        foreignKey:{{ (concatWith $rel.ThisSchemaColumnNames ",") | lowerCase}};{{- /**/ -}}
        references:{{ (concatWith $rel.OtherSchemaColumnNames ",")|lowerCase}};belongsTo;{{- /**/ -}}
        constraint:OnDelete:{{ if $rel.RestrictDelete }}RESTRICT{{ else }}CASCADE{{ end }}{{- /**/ -}}
        "`
    {{- end}}
    }
    {{- range $idx, $child := $schema.Children }}
        {{- template "createGormModel" $child }}
    {{- end }}
{{- end}}
{{- define "createTableNames" }}
	{{.Table|upperCamelCase}}TableName = "{{.Table|lowerCase}}"
	{{- range $idx, $child := .Children }}
	   {{- template "createTableNames" $child }}
    {{- end }}
{{- end}}

const (
    {{- template "createTableNames" .Schema }}
)

{{- template "createGormModel" .Schema }}
