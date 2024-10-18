{{/* The header of the generated file */}}
package customresource

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
	"google.golang.org/protobuf/types/known/timestamppb"
)

{{- range .TypesToConvert }}
// {{.TypeName}} represents storage.{{.TypeName}} in the Custom Resource.
type {{.TypeName}} struct {
     {{- range .Fields }}
        {{- if .NeedConversion }}
        {{.Name}} {{.TrimmedType}}{{if .YamlTag}} `yaml:"{{.YamlTag}}"`{{end}}
        {{- else }}
        {{.Name}} {{.Type}}{{if .YamlTag}}  `yaml:"{{.YamlTag}}"`{{end}}
        {{- end}}
     {{- end }}
}

// convert{{.TypeName}} Converts storage.{{.TypeName}} to *{{.TypeName}}
func convert{{.TypeName}}(p *storage.{{.TypeName}}) *{{.TypeName}} {
	if p == nil {
		return nil
	}

	return &{{.TypeName}}{
         {{- range .Fields }}
         {{- $fieldName := printf "p.%s" .Name }}
         {{- if .IsTimestamp }}
             {{- $fieldName = printf "timestampToFormatRFC3339(%s)" $fieldName }}
         {{- end }}
         {{ .Name }}:
         {{- if .IsStringer -}}
             {{- if .IsSlice -}}
                 sliceutils.StringSlice({{ $fieldName }}...)
             {{- else -}}
                 {{ $fieldName }}.String()
             {{- end -}}
         {{- else if .NeedConversion -}}
             {{- if .IsSlice  -}}
                 sliceutils.ConvertSlice({{ $fieldName }}, convert{{ .BaseType }})
             {{- else -}}
                 convert{{ .BaseType }}({{ $fieldName }})
             {{- end -}}
         {{- else -}}
             {{- $fieldName -}}
         {{- end }},
         {{- end }}
	}
}
{{- end }}

// Convert{{.TypeName}}ToCustomResource converts a storage.{{.TypeName}} to a SecurityPolicy custom resource
func Convert{{.TypeName}}ToCustomResource(p *storage.{{.TypeName}}) *CustomResource {
	if p == nil {
		return nil
	}
    return &CustomResource{
        APIVersion: "config.stackrox.io/v1alpha1",
        Kind:       "SecurityPolicy",
        Metadata:   map[string]interface{}{"name": toDNSSubdomainName(p.GetName())},
        SecurityPolicySpec: convert{{.TypeName}}(p),
    }
}
