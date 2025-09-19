
package schema

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// generated{{.TypeName}}SearchFields contains pre-computed search fields for {{.Table}}
	generated{{.TypeName}}SearchFields = map[search.FieldLabel]*search.Field{
		{{- range .SearchFields }}
		search.FieldLabel("{{.FieldLabel}}"): {
			FieldPath: "{{.FieldPath}}",
			Store:     {{.Store}},
			Hidden:    {{.Hidden}},
			Category:  []v1.SearchCategory{v1.{{.SearchCategory}}},
			{{- if .Analyzer }}
			Analyzer:  search.{{.Analyzer}},
			{{- end }}
		},
		{{- end }}
	}

	// generated{{.TypeName}}Schema is the pre-computed schema for {{.Table}} table
	generated{{.TypeName}}Schema = &walker.Schema{
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
				DataType:   postgres.{{.DataType}},
				{{- if .IsPrimaryKey }}
				Options: walker.PostgresOptions{
					PrimaryKey: true,
				},
				{{- end }}
			},
			{{- end }}
		},
	}
)

// Get{{.TypeName}}Schema returns the generated schema for {{.Table}}
func Get{{.TypeName}}Schema() *walker.Schema {
	// Set up search options if not already done
	if generated{{.TypeName}}Schema.OptionsMap == nil {
		generated{{.TypeName}}Schema.SetOptionsMap(search.OptionsMapFromMap(v1.{{.SearchCategory}}, generated{{.TypeName}}SearchFields))
	}
	return generated{{.TypeName}}Schema
}