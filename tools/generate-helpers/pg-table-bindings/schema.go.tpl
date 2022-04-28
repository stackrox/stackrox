{{define "schemaVar"}}{{.Table|upperCamelCase}}Schema{{end}}
{{define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}

package schema

import (
    "context"

    "github.com/stackrox/rox/central/globaldb"
    pkgSchema "github.com/stackrox/rox/central/postgres/schema"
    "github.com/stackrox/rox/generated/storage"
    "github.com/stackrox/rox/pkg/search"
)

{{- define "createTableStmt" }}
{{- $schema := . }}
&postgres.CreateStmts{
    Table: `
               create table if not exists {{$schema.Table}} (
               {{- range $idx, $field := $schema.ResolvedFields }}
                   {{$field.ColumnName}} {{$field.SQLType}}{{if $field.Options.Unique}} UNIQUE{{end}},
               {{- end}}
                   PRIMARY KEY({{template "commaSeparatedColumns" $schema.ResolvedPrimaryKeys }})
               {{- range $idx, $pksGrps := $schema.ReferenceKeysGroupedByTable -}},
                   CONSTRAINT fk_parent_table_{{$idx}} FOREIGN KEY ({{template "commaSeparatedColumns" $pksGrps.Fields}}) REFERENCES {{$pksGrps.Table}}({{template "commandSeparatedRefs" $pksGrps.Fields}}) ON DELETE CASCADE
               {{- end}}
               )
               `,
    Indexes: []string {
                   {{- range $idx, $field := $schema.Fields}}
                       {{if $field.Options.Index}}"create index if not exists {{$schema.Table|lowerCamelCase}}_{{$field.ColumnName}} on {{$schema.Table}} using {{$field.Options.Index}}({{$field.ColumnName}})",{{end}}
                   {{- end}}
                   },
    Children: []*postgres.CreateStmts{
     {{- range $idx, $child := $schema.ReferencingSchema }}
            {{- if eq $child.EmbeddedIn $schema.Table }}
                {{- template "createTableStmt" $child }},
            {{- end }}
        {{- end }}
    },
}
{{- end}}

var (
    // {{template "createTableStmtVar" .Schema }} holds the create statement for table `{{.Schema.Table|lowerCase}}`.
    {{template "createTableStmtVar" .Schema }} = {{template "createTableStmt" .Schema }}

    // {{template "schemaVar" .Schema}} is the go schema for table `{{.Schema.Table|lowerCase}}`.
    {{template "schemaVar" .Schema}} = func() *walker.Schema {
        schema := globaldb.GetSchemaForTable("{{.Schema.Table}}")
        if schema != nil {
            return schema
        }
        schema = walker.Walk(reflect.TypeOf(({{.Schema.Type}})(nil)), "{{.Schema.Table}}")
        {{- /* Attach reference schemas, if provided. */ -}}
        {{- $schema := .Schema }}
        {{- range $idx, $ref := .Refs}}.
            WithReference({{template "schemaVar" $ref}})
        {{- end }}
        {{- if .SearchCategory }}
            {{- $ty := .Schema.Type }}
            {{- /* TODO: [ROX-10206] Reconcile storage.ListAlert search terms with storage.Alert */ -}}
            {{- if eq $ty "*storage.Alert"}}
                {{- $ty = "*storage.ListAlert"}}
            {{- end}}
            schema.SetOptionsMap(search.Walk({{.SearchCategory}}, "{{.Schema.Table}}", ({{$ty}})(nil)))
        {{- end }}
        globaldb.RegisterTable(schema)
        return schema
    }()
)
