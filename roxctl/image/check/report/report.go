package report

import (
	"encoding/json"
	"io"
	"strings"
	"text/template"

	"github.com/mitchellh/go-wordwrap"
	"github.com/stackrox/rox/generated/storage"
)

const (
	// failedTemplate is a (raw) template for displaying when there are failed
	// policies.
	failedTemplate = `----------------BEGIN STACKROX CI---------------
The scanned image violated the following policies:
{{- range .}}
✗ {{.Name}}{{if failed .}} (policy enforcement failed build){{end -}}
{{if .Description}}
- Description:
    ↳ {{wrap .Description}}{{end -}}
{{if .Rationale}}
- Rationale:
    ↳ {{wrap .Rationale}}{{end -}}
{{if .Remediation}}
- Remediation:
    ↳ {{wrap .Remediation}}{{end -}}
{{if .Categories}}
- Categories:
    ↳ {{join .Categories ", "}}{{end -}}
{{- end}}
----------------END STACKROX CI---------------
`

	// passedTemplate is a (raw) template for displaying when there are no
	// failed policies.
	passedTemplate = `----------------BEGIN STACKROX CI---------------
✔ The scanned image passed all policies
----------------END STACKROX CI---------------
`
)

var (
	// failedTpl is a (parsed) template for displaying when there are failed
	// policies.
	failedTpl = template.Must(template.New("failed").Funcs(
		template.FuncMap{
			"failed": EnforcementFailedBuild,
			"join":   strings.Join,
			"wrap":   wrap,
		},
	).Parse(failedTemplate))

	// passedTpl is a (parsed) template for displaying when there are no failed
	// policies.
	passedTpl = template.Must(template.New("passed").Parse(passedTemplate))
)

// JSON renders the given list of policies as JSON, and writes that to the
// output stream.
func JSON(output io.Writer, policies []*storage.Policy) error {
	// Just pipe out the violated policies as JSON.
	body, err := json.MarshalIndent(policies, "", "  ")
	if err != nil {
		return err
	}
	output.Write(body)
	output.Write([]byte{'\n'})
	return nil
}

// Pretty renders the given list of policies in a human-friendly format, and
// writes that to the output stream.
func Pretty(output io.Writer, policies []*storage.Policy) error {
	switch len(policies) {
	case 0:
		return passedTpl.Execute(output, policies)
	default:
		return failedTpl.Execute(output, policies)
	}
}

// EnforcementFailedBuild returns true if the given policy has an enforcement
// action that fails the CI build. Intended to be uses as a test template
// function.
func EnforcementFailedBuild(policy *storage.Policy) bool {
	for _, action := range policy.GetEnforcementActions() {
		if action == storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT {
			return true
		}
	}
	return false
}

// wrap performs line-wrapping of the given text at 80 characters in length.
// Intended to be uses as a test template function.
func wrap(text string) string {
	wrapped := wordwrap.WrapString(text, 80)
	wrapped = strings.TrimSpace(wrapped)
	wrapped = strings.Replace(wrapped, "\n", "\n      ", -1)
	return wrapped
}
