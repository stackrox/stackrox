{{define "schemaVar"}}{{.|upperCamelCase}}Schema{{end}}
{{define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{define "commmaSeparatedColumnNamesFromField"}}{{range $idx, $field:= .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commaSeparatedColumnsInThisTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInThisSchema}}{{end}}{{end}}
{{define "commaSeparatedColumnsInOtherTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInOtherSchema}}{{end}}{{end}}

package schema

import (
    "context"
    "fmt"

    "github.com/stackrox/rox/central/globaldb"
    v1 "github.com/stackrox/rox/generated/api/v1"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/postgres"
    "github.com/stackrox/rox/pkg/postgres/walker"
    "github.com/stackrox/rox/pkg/search"
)

{{- define "createTableStmt" }}
{{- $schema := . }}
&postgres.CreateStmts{
    Table: `
               create table if not exists {{$schema.Table}} (
               {{- range $idx, $field := $schema.DBColumnFields }}
                   {{$field.ColumnName}} {{$field.SQLType}}{{if $field.Options.Unique}} UNIQUE{{end}},
               {{- end}}
                   PRIMARY KEY({{template "commmaSeparatedColumnNamesFromField" $schema.PrimaryKeys }})
               {{- range $idx, $rel := $schema.RelationshipsToDefineAsForeignKeys -}},
                   CONSTRAINT fk_parent_table_{{$idx}} FOREIGN KEY ({{template "commaSeparatedColumnsInThisTable" $rel.MappedColumnNames}}) REFERENCES {{$rel.OtherSchema.Table}}({{template "commaSeparatedColumnsInOtherTable" $rel.MappedColumnNames}}) ON DELETE CASCADE
               {{- end}}
               )
               `,
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
        schema := globaldb.GetSchemaForTable("{{.Schema.Table}}")
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
            schema.SetOptionsMap(search.Walk(v1.{{.SearchCategory}}, "{{.Schema.Table}}", ({{$ty}})(nil)))
        {{- end }}
        globaldb.RegisterTable(schema)
        return schema
    }()
)
