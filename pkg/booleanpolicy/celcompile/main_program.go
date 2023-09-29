package celcompile

import (
	"strings"
	"text/template"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log                 = logging.LoggerForModule()
	mainProgramTemplate = template.Must(template.New("").Parse(
		`
{{- define "valueMatch" }}
  {{- $desc := . }}
.map(rs, {{$desc.MatchCode}}, rs.map(r, r.with({"{{$desc.SearchName}}": [{{$desc.VarName}}]})))
{{- end}}

{{- define "arrayValueMatch" }}
  {{- $desc := . }}
  .map(
    prevResults,
    {{$desc.CheckCode}},
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
	     .map(rs, prevResults.map(p, rs.map(r, p.with(r))))
	  )
      .flatten()
  )
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
