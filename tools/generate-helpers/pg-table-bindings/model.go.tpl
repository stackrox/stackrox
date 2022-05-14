{{define "commaSeparatedColumns"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commandSeparatedRefs"}}{{range $idx, $field := .}}{{if $idx}}, {{end}}{{$field.Reference}}{{end}}{{end}}
{{define "commaSeparatedColumnsInThisTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInThisSchema}}{{end}}{{end}}
{{define "commaSeparatedColumnsInOtherTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInOtherSchema}}{{end}}{{end}}
package models

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

const (
	{{.Schema.Table|upperCamelCase}}TableName = "{{.Schema.Table}}"
	{{- range $idx, $child := .Schema.Children }}
       {{$child.Table|upperCamelCase}}TableName = "{{$child.Table}}"
    {{- end }}
)


{{- define "createGormModel" }}
{{- $schema := . }}
    // {{$schema.TypeName}} holds the Gorm model for Postgres table `{{$schema.Table}}`.
    type {{$schema.Table|upperCamelCase}} struct {
    {{- range $idx, $field := $schema.DBColumnFields }}
        {{$field.ColumnName}} {{$field.ModelType}} `gorm:"column:{{$field.ColumnName|lowerCase}};type:{{$field.SQLType}}{{if $field.Options.Unique}};unique{{end}}{{if $field.Options.PrimaryKey}};primaryKey{{end}}{{if $field.Options.Index}};index:{{$schema.Table|lowerCamelCase}}_{{$field.ColumnName}},type:{{$field.Options.Index}}{{end}}"`
    {{- end}}
    {{- range $idx, $rel := $schema.RelationshipsToDefineAsForeignKeys }}
        {{$rel.OtherSchema.Table|upperCamelCase}}Ref {{$rel.OtherSchema.Table|upperCamelCase}} `gorm:"foreignKey:{{template "commaSeparatedColumnsInThisTable" $rel.MappedColumnNames}};references:{{template "commaSeparatedColumnsInOtherTable" $rel.MappedColumnNames}};constraint:OnDelete:CASCADE"`
    {{- end}}
}
{{- end}}

{{- template "createGormModel" .Schema }}

{{- range $idx, $child := .Schema.Children }}
   {{- template "createGormModel" $child }}
{{- end }}