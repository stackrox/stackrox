{{- define "convertField" }}
    {{- $field := . }}
    {{- if eq $field.DataType "datetime" -}}
    protocompat.NilOrTime({{$field.Getter "obj"}}),
    {{- else -}}{{if eq $field.DataType "stringarray" -}}
    pq.Array({{$field.Getter "obj"}}).(*pq.StringArray),
    {{- else -}}{{if eq $field.DataType "enumarray" -}}
    pq.Array(pgutils.ConvertEnumSliceToIntArray({{$field.Getter "obj"}})).(*pq.Int32Array),
    {{- else -}}{{if eq $field.DataType "intarray" -}}
    pq.Array({{$field.Getter "obj"}}).(*pq.Int32Array),
    {{- else -}}
    {{$field.Getter "obj"}},
    {{- end }}{{end}}{{end}}{{end -}}
{{- end}}

{{- define "convertProtoToModel" }}
{{- $schema := . }}
    // Convert{{$schema.TypeName}}FromProto converts a `{{$schema.Type}}` to Gorm model
    func Convert{{$schema.TypeName}}FromProto(obj {{$schema.Type}}{{if $schema.Parent}}, idx int{{end}}{{ range $index, $field := $schema.FieldsReferringToParent }}, {{$field.Name}} {{$field.Type}}{{end}}) (*{{$schema.Table|upperCamelCase}}, error) {
        {{- if not $schema.Parent }}
        serialized, err := obj.MarshalVT()
        if err != nil {
            return nil, err
        }
        {{- end}}
        model := &{{$schema.Table|upperCamelCase}}{
        {{- range $index, $field := $schema.DBColumnFields }}
            {{$field.ColumnName|upperCamelCase}}: {{- template "convertField" $field}}
        {{- end}}
        }
        return model, nil
    }

    {{- range $index, $child := $schema.Children }}
        {{- template "convertProtoToModel" $child }}
    {{- end }}

    {{- if not $schema.Parent }}
    // Convert{{$schema.TypeName}}ToProto converts Gorm model `{{$schema.Table|upperCamelCase}}` to its protobuf type object
    func Convert{{$schema.TypeName}}ToProto(m *{{$schema.Table|upperCamelCase}}) ({{$schema.Type}}, error) {
        var msg storage.{{$schema.TypeName}}
        if err := msg.UnmarshalVTUnsafe(m.Serialized); err != nil {
            return nil, err
        }
        return &msg, nil
    }
    {{- end }}
{{- end}}
package schema

import (
	"github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protocompat"
)

{{- template "convertProtoToModel" .Schema }}
