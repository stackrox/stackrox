{{define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}

package schema

import (
     "github.com/stackrox/rox/pkg/postgres"
)

{{- define "createTableStmt" }}
{{- $schema := . }}
&postgres.CreateStmts{
    Table: `
               create table if not exists {{$schema.Table}} (
               {{- range $idx, $field := $schema.ResolvedFields }}
                   {{$field.ColumnName}} {{$field.SQLType}}{{if $field.Options.Unique}} UNIQUE{{end}},
               {{- end}}
                   PRIMARY KEY({{template "commaSeparatedColumns" $schema.ResolvedPrimaryKeys }}){{ if gt (len $schema.Parents) 0 }},{{end}}
               {{- range $idx, $pksGrps := $schema.ParentKeysGroupedByTable }}
                   CONSTRAINT fk_parent_table_{{$idx}} FOREIGN KEY ({{template "commaSeparatedColumns" $pksGrps.Fields}}) REFERENCES {{$pksGrps.Table}}({{template "commandSeparatedRefs" $pksGrps.Fields}}) ON DELETE CASCADE{{if lt (add $idx 1) (len $schema.ParentKeysGroupedByTable)}},{{end}}
               {{- end}}
               )
               `,
    Indexes: []string {
                   {{- range $idx, $field := $schema.Fields}}
                       {{if $field.Options.Index}}"create index if not exists {{$schema.Table|lowerCamelCase}}_{{$field.ColumnName}} on {{$schema.Table}} using {{$field.Options.Index}}({{$field.ColumnName}})",{{end}}
                   {{- end}}
                   },
    Children: []*postgres.CreateStmts{
     {{- range $idx, $child := $schema.Children }}
            {{- if eq $child.EmbeddedIn $schema.Table }}
                {{- template "createTableStmt" $child }},
            {{- end }}
        {{- end }}
    },
}
{{- end}}

var (
   // {{template "createTableStmtVar" .Schema }} holds the create statement for table `{{.Schema.Table|upperCamelCase}}`.
   {{template "createTableStmtVar" .Schema }} = {{template "createTableStmt" .Schema }}
)
