package celcompile

import (
	"strings"
	"text/template"
)

/*
var tplate2 = `
[[{}]]

	.map(result, obj.ValA.startsWith("TopLevelValA"), result.map(t, t.with({"TopLevelValA": [obj.ValA]})))
	.map(
	   result,
	   obj.NestedSlice
	     .filter(
	       k,
	       k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2")
	     )
	     .filter(
	       k,
	       k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2")
	     )
	     .map(
	       k,
	       result.map(entry, entry.with({"B": [k.NestedValB], "A": [k.NestedValA]}))
	     ).flatten()
	)
	.filter(result, result.size() != 0)

`
*/
var (
	mainProgramTemplate = template.Must(template.New("").Parse(
		/*` IndexesToDeclare, Functions, Conditions
		{{ $root := . }}

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
					{{- if $index }},{{ end }}
					"{{ $field.Name }}": {{ $field.FuncName }}Result["values"]
				{{- end }}
			}
		}
		`*/
		`
{{ $root := . }}

[]
{{- range .Conditions }}
+[[{}]]
    //0 index
    //filterClause = obj.ValA.startsWith("TopLevelValA")
    //resultClause = result.map(t, t.with({"TopLevelValA": [obj.ValA]}))
   .map(
     result,
     obj.ValA.startsWith("TopLevelValA"),
     result.map(t, t.with({"TopLevelValA": [obj.ValA]}))
   )
    //1 index
   .map(
      result,
      obj.NestedSlice
        .filter(
          k,
          k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2")
        )
        .filter(
          k,
          k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2")
        )
        .map(
          k,
          result.map(entry, entry.with({"B": [k.NestedValB], "A": [k.NestedValA]}))
        ).flatten()
   )
   .filter(result, result.size() != 0)
   .flatten()
{{- end }}
`,
	))
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
