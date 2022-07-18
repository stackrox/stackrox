{{define "schemaVar"}}{{.|upperCamelCase}}Schema{{end}}
{{define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{define "commmaSeparatedColumnNamesFromField"}}{{range $idx, $field:= .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commaSeparatedColumnsInThisTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInThisSchema}}{{end}}{{end}}
{{define "commaSeparatedColumnsInOtherTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInOtherSchema}}{{end}}{{end}}

package schema

import (
    "context"
    "fmt"

    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/postgres/registry"
    "github.com/stackrox/rox/pkg/postgres/walker"
    "github.com/stackrox/rox/pkg/search"
)

{{- define "createTableStmt" }}
{{- $schema := . }}
&postgres.CreateStmts{
    Table: `
               create table if not exists {{$schema.Table}} (
               {{- range $idx, $field := $schema.DBColumnFields }}
                   {{$field.ColumnName}} {{$field.SQLType}}{{if $field.Options.Unique}} UNIQUE{{end}}{{ if $schema.PrimaryKeys }},{{end}}
               {{- end}}
                   {{ if $schema.PrimaryKeys }}PRIMARY KEY({{template "commmaSeparatedColumnNamesFromField" $schema.PrimaryKeys }}){{end}}
               {{- range $idx, $rel := $schema.RelationshipsToDefineAsForeignKeys -}},
                   CONSTRAINT fk_parent_table_{{$idx}} FOREIGN KEY ({{template "commaSeparatedColumnsInThisTable" $rel.MappedColumnNames}}) REFERENCES {{$rel.OtherSchema.Table}}({{template "commaSeparatedColumnsInOtherTable" $rel.MappedColumnNames}}) ON DELETE CASCADE
               {{- end}}
               )
               `,
    GormModel: (*{{$schema.Table|upperCamelCase}})(nil),
    Indexes: []string {
                   {{- range $idx, $field := $schema.Fields}}
                       {{- if $field.Options.Index}}
                       "create index if not exists {{$schema.Table|lowerCamelCase}}_{{$field.ColumnName}} on {{$schema.Table}} using {{$field.Options.Index}}({{$field.ColumnName}})",
                       {{- end}}
                   {{- end}}
                   },
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
        schema := registry.GetSchemaForTable("{{.Schema.Table}}")
        if schema != nil {
            return schema
        }
        schema = walker.Walk(reflect.TypeOf(({{.Schema.Type}})(nil)), "{{.Schema.Table}}")

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
            {{- $ty := .Schema.Type }}
            {{- /* TODO: [ROX-10206] Reconcile storage.ListAlert search terms with storage.Alert */ -}}
            {{- if eq $ty "*storage.Alert"}}
                {{- $ty = "*storage.ListAlert"}}
            {{- end}}
            schema.SetOptionsMap(search.Walk(v1.{{.SearchCategory}}, "{{.Schema.TypeName|lower}}", ({{$ty}})(nil)))
            {{- if .SearchScope }}
            schema.SetSearchScope([]v1.SearchCategory{
            {{- range $category := .SearchScope }}
                {{$category}},
            {{- end}}
            }...)
            {{- end }}
        {{- end }}
        registry.RegisterTable(schema, {{template "createTableStmtVar" .Schema }})
        return schema
    }()
)

{{- define "createGormModel" }}
{{- $schema := . }}
    // {{$schema.Table|upperCamelCase}} holds the Gorm model for Postgres table `{{$schema.Table|lowerCase}}`.
    type {{$schema.Table|upperCamelCase}} struct {
    {{- range $idx, $field := $schema.DBColumnFields }}
        {{$field.ColumnName|upperCamelCase}} {{$field.ModelType}} `gorm:"column:{{$field.ColumnName|lowerCase}};type:{{$field.SQLType}}{{if $field.Options.Unique}};unique{{end}}{{if $field.Options.PrimaryKey}};primaryKey{{end}}{{if $field.Options.Index}};index:{{$schema.Table|lowerCamelCase|lowerCase}}_{{$field.ColumnName|lowerCase}},type:{{$field.Options.Index}}{{end}}"`
    {{- end}}
    {{- range $idx, $rel := $schema.RelationshipsToDefineAsForeignKeys }}
        {{$rel.OtherSchema.Table|upperCamelCase}}Ref {{$rel.OtherSchema.Table|upperCamelCase}} `gorm:"foreignKey:{{ (concatWith $rel.ThisSchemaColumnNames ",") | lowerCase}};references:{{ (concatWith $rel.OtherSchemaColumnNames ",")|lowerCase}};belongsTo;constraint:OnDelete:CASCADE"`
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
