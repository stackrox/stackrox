package report

import (
	"io"
	"text/template"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// failedTemplate is a (raw) template for displaying when there are failed
	// policies.
	failedTemplate = `{{- range .Alerts}}
{{if .Deployment}}
✗ Deployment {{.Deployment.Name}} failed policy '{{.Policy.Name}}' {{if failedFunc .Policy}}(policy enforcement caused failure){{end}}
{{else}}
✗ Image {{$.ResourceName}} failed policy '{{.Policy.Name}}' {{if failedFunc .Policy}}(policy enforcement caused failure){{end}}
{{end -}}
- Description:
    ↳ {{wrap .Policy.Description}}
- Rationale:
    ↳ {{wrap .Policy.Rationale}}
- Remediation:
    ↳ {{wrap .Policy.Remediation}}
- Violations:
    {{- range .Violations}}
    - {{.Message}}
    {{- end}}
{{- end}}

`

	// passedTemplate is a (raw) template for displaying when there are no
	// failed policies.
	passedTemplate = `✔ The scanned resources passed all policies
`
)

// JSON renders the given list of policies as JSON, and writes that to the
// output stream.
func JSON(output io.Writer, alerts []*storage.Alert) error {
	// Just pipe out the violated alerts as JSON.

	// This object is really just a filler because its a wrapper around the alerts
	// this is required because jsonpb can only marshal proto.Message
	bdr := &v1.BuildDetectionResponse{
		Alerts: alerts,
	}
	marshaler := jsonpb.Marshaler{Indent: "  "}
	if err := marshaler.Marshal(output, bdr); err != nil {
		return err
	}
	if _, err := output.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}

// PrettyWithResourceName renders the given list of policies in a human-friendly format, and
// writes that to the output stream.
func PrettyWithResourceName(output io.Writer, alerts []*storage.Alert, enforcementStage storage.EnforcementAction, resourceType, resourceName string) error {
	var templateMap = map[string]interface{}{
		"Alerts":       alerts,
		"ResourceType": resourceType,
		"ResourceName": resourceName,
	}

	funcMap := template.FuncMap{
		"failedFunc": EnforcementFailedBuild(enforcementStage),
		"wrap":       stringutils.Wrap,
	}

	var t *template.Template
	if len(alerts) == 0 {
		t = template.Must(template.New("passed").Funcs(funcMap).Parse(passedTemplate))
	} else {
		t = template.Must(template.New("failed").Funcs(funcMap).Parse(failedTemplate))
	}
	return t.Execute(output, templateMap)
}

// Pretty is a wrapper around PrettyWithResourceName that gets called with an empty resource name
func Pretty(output io.Writer, alerts []*storage.Alert, enforcementStage storage.EnforcementAction, resourceType string) error {
	return PrettyWithResourceName(output, alerts, enforcementStage, resourceType, "")
}

// EnforcementFailedBuild returns true if the given policy has an enforcement
// action that fails the CI build. Intended to be uses as a test template
// function.
func EnforcementFailedBuild(enforcementAction storage.EnforcementAction) func(policy *storage.Policy) bool {
	return func(policy *storage.Policy) bool {
		for _, action := range policy.GetEnforcementActions() {
			if action == enforcementAction {
				return true
			}
		}
		return false
	}
}
