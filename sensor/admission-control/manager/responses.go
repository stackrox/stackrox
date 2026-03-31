package manager

import (
	"fmt"
	"text/template"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/templates"
	admission "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	kubectlTemplate = `
{{- range .Alerts -}}
Policy: {{.Policy.Name}}
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

{{ end -}}
{{- if gt .UnevaluatedPolicyCount 0}}
{{.UnevaluatedPolicyCount}} additional {{if eq .UnevaluatedPolicyCount 1}}policy depends{{else}}policies depend{{end}} on image enrichment results and will be evaluated only after the above violations are addressed.
{{ end -}}
{{- if .BypassAnnotationKey}}
In case of emergency, add the annotation {"{{.BypassAnnotationKey}}": "ticket-1234"} to your deployment with an updated ticket number
{{- end -}}
`
)

var (
	msgTemplate = template.Must(template.New("name").Funcs(
		template.FuncMap{
			"wrap": stringutils.Wrap,
		}).Parse(kubectlTemplate))
)

func pass(uid types.UID) *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		UID:     uid,
		Allowed: true,
	}
}

func fail(uid types.UID, message string) *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		UID:     uid,
		Allowed: false,
		Result: &metav1.Status{
			Status:  "Failure",
			Reason:  metav1.StatusReason(fmt.Sprintf("Failed currently enforced policies from %s", branding.GetProductNameShort())),
			Message: message,
		},
	}
}

func message(alerts []*storage.Alert, addBypassMsg bool, unevaluatedPolicyCount int) string {
	// We add a line break at the beginning to look nicer in kubectl
	msgHeader := "\nThe attempted operation violated one or more enforced policies, described below:\n\n"
	data := map[string]interface{}{
		"Alerts":                 alerts,
		"UnevaluatedPolicyCount": unevaluatedPolicyCount,
	}

	if addBypassMsg {
		data["BypassAnnotationKey"] = enforcers.EnforcementBypassAnnotationKey
	}

	msgBody, err := templates.ExecuteToString(msgTemplate, data)
	if err != nil {
		msgBody = fmt.Sprintf("Internal error executing message template: %v", err)
	}

	return msgHeader + msgBody + "\n\n"
}
