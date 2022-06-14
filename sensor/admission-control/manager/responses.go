package manager

import (
	"fmt"
	"text/template"

	"github.com/stackrox/rox/generated/storage"
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
			Reason:  metav1.StatusReason("Failed currently enforced policies from StackRox"),
			Message: message,
		},
	}
}

func message(alerts []*storage.Alert, addBypassMsg bool) string {
	noun := "policies"
	if len(alerts) == 1 {
		noun = "policy"
	}

	// We add a line break at the beginning to look nicer in kubectl
	msgHeader := fmt.Sprintf("\nThe attempted operation violated %d enforced %s, described below:\n\n", len(alerts), noun)
	data := map[string]interface{}{
		"Alerts": alerts,
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
