
package internal

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// {{.TypeName}}SearchFields contains pre-computed search fields for {{.Table}}
	{{.TypeName}}SearchFields = map[search.FieldLabel]*search.Field{
		{{- range .SearchFields }}
		search.FieldLabel("{{.FieldLabel}}"): {
			FieldPath: "{{.FieldPath}}",
			Store:     {{.Store}},
			Hidden:    {{.Hidden}},
			{{- if .SearchCategory }}
			Category:  v1.{{.SearchCategory}},
			{{- end }}
			{{- if .Analyzer }}
			Analyzer:  "{{.Analyzer}}",
			{{- end }}
		},
		{{- end }}
	}

	// {{.TypeName}}Schema is the pre-computed schema for {{.Table}} table
	{{.TypeName}}Schema = &walker.Schema{
		Table:    "{{.Table}}",
		Type:     "{{.Type}}",
		TypeName: "{{.TypeName}}",
		Fields: []walker.Field{
			{{- range .Fields }}
			{
				Name:       "{{.Name}}",
				ColumnName: "{{.ColumnName}}",
				Type:       "{{.Type}}",
				SQLType:    "{{.SQLType}}",
				{{- if .DataType }}
				DataType:   postgres.{{.DataType}},
				{{- end }}
				{{- if .SearchFieldName }}
				Search: walker.SearchField{
					FieldName: "{{.SearchFieldName}}",
					Enabled:   true,
				},
				{{- end }}
				{{- if .IsPrimaryKey }}
				Options: walker.PostgresOptions{
					PrimaryKey: true,
				},
				{{- end }}
			},
			{{- end }}
		},
		{{- if .ChildSchemas }}
		Children: []*walker.Schema{
			{{- range .ChildSchemas }}
			{{- template "childSchema" . }}
			{{- end }}
		},
		{{- end }}
	}
)

// Get{{.TypeName}}Schema returns the generated schema for {{.Table}}
func Get{{.TypeName}}Schema() *walker.Schema {
	// Set up search options if not already done
	if {{.TypeName}}Schema.OptionsMap == nil {
		{{- if .SearchCategory }}
		{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.{{.SearchCategory}}, {{.TypeName}}SearchFields))
		{{- else }}
		{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.SearchCategory_SEARCH_UNSET, {{.TypeName}}SearchFields))
		{{- end }}
	}
	// Set Schema back-reference on all fields
	for i := range {{.TypeName}}Schema.Fields {
		{{.TypeName}}Schema.Fields[i].Schema = {{.TypeName}}Schema
	}
	// Set Schema back-reference on all child schema fields
	var setChildSchemaReferences func(*walker.Schema)
	setChildSchemaReferences = func(schema *walker.Schema) {
		for _, child := range schema.Children {
			for i := range child.Fields {
				child.Fields[i].Schema = child
			}
			setChildSchemaReferences(child)
		}
	}
	setChildSchemaReferences({{.TypeName}}Schema)
	return {{.TypeName}}Schema
}

{{- define "childSchema" }}
			{
				Table:    "{{.Table}}",
				Type:     "{{.Type}}",
				TypeName: "{{.TypeName}}",
				Fields: []walker.Field{
					{{- range .Fields }}
					{
						Name:       "{{.Name}}",
						ColumnName: "{{.ColumnName}}",
						Type:       "{{.Type}}",
						SQLType:    "{{.SQLType}}",
						{{- if .DataType }}
						DataType:   postgres.{{.DataType}},
						{{- end }}
						{{- if .SearchFieldName }}
						Search: walker.SearchField{
							FieldName: "{{.SearchFieldName}}",
							Enabled:   true,
						},
						{{- end }}
						{{- if .IsPrimaryKey }}
						Options: walker.PostgresOptions{
							PrimaryKey: true,
						},
						{{- end }}
					},
					{{- end }}
				},
				{{- if .ChildSchemas }}
				Children: []*walker.Schema{
					{{- range .ChildSchemas }}
					{{- template "childSchema" . }}
					{{- end }}
				},
				{{- end }}
			},
{{- end }}