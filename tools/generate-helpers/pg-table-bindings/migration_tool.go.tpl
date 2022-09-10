{{define "schemaVar"}}{{.|upperCamelCase}}Schema{{end}}
{{define "createTableStmtVar"}}CreateTable{{.Table|upperCamelCase}}Stmt{{end}}
{{define "commmaSeparatedColumnNamesFromField"}}{{range $idx, $field:= .}}{{if $idx}}, {{end}}{{$field.ColumnName}}{{end}}{{end}}
{{define "commaSeparatedColumnsInThisTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInThisSchema}}{{end}}{{end}}
{{define "commaSeparatedColumnsInOtherTable"}}{{range $idx, $columnNamePair := .}}{{if $idx}}, {{end}}{{$columnNamePair.ColumnNameInOtherSchema}}{{end}}{{end}}

package convert

import (
	"github.com/gogo/protobuf/proto"
	"github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

{{- define "convertField" }}
    {{- $field := . }}
    {{- if eq $field.DataType "datetime" -}}
    pgutils.NilOrTime({{$field.Getter "obj"}}),
    {{- else -}}{{if eq $field.DataType "stringarray" -}}
    pq.Array({{$field.Getter "obj"}}).(*pq.StringArray),
    {{- else -}}{{if eq $field.DataType "enumarray" -}}
    pq.Array(pgutils.ConvertEnumSliceToIntArray({{$field.Getter "obj"}})).(*pq.Int32Array),
    {{- else -}}
    {{$field.Getter "obj"}},
    {{- end -}}{{- end -}}{{- end -}}
{{- end}}
{{- define "convertProtoToModel" }}
{{- $schema := . }}
    func convert{{$schema.TypeName}}FromProto(obj {{$schema.Type}}{{ range $idx, $field := $schema.FieldsReferringToParent }}, {{$field.Name}} {{$field.Type}}{{end}}) (*schema.{{$schema.Table|upperCamelCase}}, error) {
    	serialized, err := obj.Marshal()
    	if err != nil {
    		return nil, err
    	}
    	model := &schema.{{$schema.Table|upperCamelCase}}{
        {{- range $idx, $field := $schema.DBColumnFields }}
            {{$field.ColumnName|upperCamelCase}}: {{- template "convertField" $field}}
        {{- end}}
        }
        return model, nil
    }

    func convert{{$schema.TypeName}}ToProto(m *schema.{{$schema.Table|upperCamelCase}}) ({{$schema.Type}}, error) {
    	var msg storage.{{$schema.TypeName}}
    	if err := msg.Unmarshal(m.Serialized); err != nil {
        	return nil, err
        }
    	return &msg, nil
    }

    func Test{{$schema.TypeName}}Conversion(t *testing.T) {
    	obj := &storage.{{$schema.TypeName}}{}
    	assert.NoError(t, testutils.FullInit(obj, testutils.UniqueInitializer(), testutils.JSONFieldsFilter))
       	m, err := convert{{$schema.TypeName}}FromProto(obj)
       	assert.NoError(t, err)
       	conv, err := convert{{$schema.TypeName}}ToProto(m)
       	assert.NoError(t, err)
       	assert.Equal(t, obj, conv)
    }
    {{- range $idx, $child := $schema.Children }}
        {{- template "convertProtoToModel" $child }}
    {{- end }}
{{- end}}

{{- template "convertProtoToModel" .Schema }}
