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
		/*
			[[{}]]
			   .map(r, obj.ValA.startsWith("TopLevelValA"), r.map(t, t.with({"TopLevelA": [obj.ValA]})))
			   .map(
			      result,
			      obj.NestedSlice
			        .map(
			          k,
			          [[{}]]
			           .map(result1, k.NestedValB.startsWith("B1") || k.NestedValB.startsWith("B2"), result1.map(t, t.with({"B": [k.NestedValB]})))
			           .map(result1, k.NestedValA.startsWith("A1") || k.NestedValA.startsWith("A2"), result1.map(t, t.with({"A": [k.NestedValA]})))
			           .map(result1, result.map(t, result1.map(x, t.with(x))))
			        )
			   )
			   .filter(result, result.size() != 0)
			   .flatten()
		*/
		`
{{- define "valueMatch" }}
  {{- $desc := . }}
  .map(rs, {{$desc.MatchCode}}, [rs].flatten().map(r, r.with({"{{$desc.SearchName}}": [{{$desc.VarName}}]})))
{{- end}}
{{- define "arrayValueMatch" }}
  {{- $desc := . }}
		   .map(
		      prevResults,
              has({{$desc.VarName}}) && {{$desc.VarName}} != null,
              //{{$desc.CheckCode}},
              {{$desc.VarName}}
		        .map(
		          k,
		          [[{}]]
                   {{- range $index, $child := $desc.Children }} 
                   	 {{- if $child.IsLeaf }}
                     {{- template "valueMatch" $child }}
                     {{- else }}
                     {{- template "arrayValueMatch" $child }}
                     {{- end}}
                   {{- end}}
		           .filter(r, [r].flatten().size() != 0)
		           .map(rs, [prevResults].flatten().map(p, [rs].flatten().map(r, p.with(r))))
                    .flatten()
        		         .filter(r, [r].flatten().size() != 0)
		        )
		   )
		   .filter(r, [r].flatten().size() != 0)
{{- end}}

{{ $root := . }}

[]
+[[{}]]
{{- range $index, $child := .Root.Children }} 
 {{- if $child.IsLeaf }}
 {{- template "valueMatch" $child }}
 {{- else }}
 {{- template "arrayValueMatch" $child }}
 {{- end}}
{{- end}}
.flatten()
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

type MatchField struct {
	VarName    string
	SearchName string // Only for non-array
	MatchCode  string // Only for non-array
	IsLeaf     bool
	Path       string // Not in use now
	CheckCode  string

	Children []*MatchField
}

type mainProgramArgs struct {
	Root MatchField
}

func generateMainProgram(args *mainProgramArgs) (string, error) {
	var sb strings.Builder
	if err := mainProgramTemplate.Execute(&sb, args); err != nil {
		return "", err
	}
	return sb.String(), nil
}
