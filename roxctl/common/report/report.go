package report

import (
	"io"
	"text/template"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	// failedTemplate is a (raw) template for displaying when there are failed
	// policies.
	failedTemplate = `{{- range .AlertTemplateObjects}}
{{if .Alert.GetDeployment}}
✗ Deployment {{.Alert.GetDeployment.Name}} failed policy '{{.Alert.Policy.Name}}'{{if failedFunc .Alert.Policy}} (policy enforcement caused failure){{end}}
{{else}}
✗ Image {{$.ResourceName}} failed policy '{{.Alert.Policy.Name}}'{{if failedFunc .Alert.Policy}} (policy enforcement caused failure){{end}}
{{end -}}
- Description:
    ↳ {{wrap .Alert.Policy.Description}}
- Rationale:
    ↳ {{wrap .Alert.Policy.Rationale}}
- Remediation:
    ↳ {{wrap .Alert.Policy.Remediation}}
- Violations:
    {{- range .PrintedViolations}}
    - {{.Message}}
    {{- end}}{{if gt .RemainingViolations 0}}
      +{{.RemainingViolations}} more{{end}}
{{- end}}

`

	// passedTemplate is a (raw) template for displaying when there are no
	// failed policies.
	//#nosec G101 -- This is a false positive
	passedTemplate = `✔ The scanned resources passed all policies
`

	maxPrintedViolations = 20
)

type alertTemplateObject struct {
	Alert               *storage.Alert
	PrintedViolations   []*storage.Alert_Violation
	RemainingViolations int
}

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
		return errors.Wrap(err, "could not marshal alerts")
	}
	if _, err := output.Write([]byte{'\n'}); err != nil {
		return errors.Wrap(err, "could not write alerts")
	}
	return nil
}

// PrettyWithResourceName renders the given list of policies in a human-friendly format, and
// writes that to the output stream.
func PrettyWithResourceName(output io.Writer, alerts []*storage.Alert, enforcementStage storage.EnforcementAction, resourceType, resourceName string, printAllViolations bool) error {
	alertTemplateObjects := makeAlertTemplateObjects(alerts, printAllViolations)
	var templateMap = map[string]interface{}{
		"AlertTemplateObjects": alertTemplateObjects,
		"ResourceType":         resourceType,
		"ResourceName":         resourceName,
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
	return errors.Wrap(t.Execute(output, templateMap), "could not render alert policies")
}

// Pretty is a wrapper around PrettyWithResourceName that gets called with an empty resource name
func Pretty(output io.Writer, alerts []*storage.Alert, enforcementStage storage.EnforcementAction, resourceType string, printAllViolations bool) error {
	return PrettyWithResourceName(output, alerts, enforcementStage, resourceType, "", printAllViolations)
}

// EnforcementFailedBuild returns a function which returns true if the given policy has an enforcement
// action that fails the CI build. Intended to be used as a test template
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

func makeAlertTemplateObjects(alerts []*storage.Alert, printAllViolations bool) []*alertTemplateObject {
	alertTemplateObjects := make([]*alertTemplateObject, len(alerts))
	for i, alert := range alerts {
		printedViolations := alert.GetViolations()
		if !printAllViolations && len(printedViolations) > maxPrintedViolations {
			printedViolations = printedViolations[:maxPrintedViolations]
		}
		alertTemplateObjects[i] = &alertTemplateObject{
			Alert:               alert,
			PrintedViolations:   printedViolations,
			RemainingViolations: len(alert.GetViolations()) - len(printedViolations),
		}
	}
	return alertTemplateObjects
}
