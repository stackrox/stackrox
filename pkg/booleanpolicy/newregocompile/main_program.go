package newregocompile

import (
	"strings"
	"text/template"
)

var (
	mainProgramTemplate = template.Must(template.New("").Parse(`
package policy.main

import future.keywords.in

negate(val) {
	some m in [val]
	not m
}

or(vals) {
	some m in vals
	m
}

{{- $root := . }}

{{- range .Functions }}
{{.}}
{{- end }}

{{- range .Conditions }}
violations[result] {
	{{- range $root.IndexesToDeclare }}
	some idx{{.}}
	{{- end }}
	{{- range .Fields }}
	{{.FuncName}}Result := {{ .FuncName }}(input.{{ .JSONPath }}) 
	{{.FuncName}}Result["match"]
	{{- end }}
	result := {
		{{- range $index, $field := .Fields }}
			{{- if $index }},{{end }} 
			"{{ $field.Name }}": {{ $field.FuncName }}Result["values"]
		{{- end }}
	}
}
{{- end }}
`))
)

type fieldInCondition struct {
	Name     string
	FuncName string
	JSONPath string
}

type condition struct {
	Fields []fieldInCondition
}

type mainProgramArgs struct {
	IndexesToDeclare []int
	Functions        []string
	Conditions       []condition
}

func generateMainProgram(args *mainProgramArgs) (string, error) {
	var sb strings.Builder
	if err := mainProgramTemplate.Execute(&sb, args); err != nil {
		return "", err
	}
	return sb.String(), nil
}
