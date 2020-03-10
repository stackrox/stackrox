package manager

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/enforcers"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/protoconv/resources"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/templates"
	admission "k8s.io/api/admission/v1beta1"
)

var (
	detectionCtx = deploytime.DetectionContext{
		EnforcementOnly: true,
	}

	builder = matcher.NewBuilder(
		matcher.NewRegistry(
			nil,
		),
		deployments.OptionsMap,
	)
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

func (m *manager) shouldBypass(s *state, req *admission.AdmissionRequest) bool {
	// If enforcement is disabled (or there are no enforced policies), mark this as pass.
	if s.detector == nil {
		log.Debugf("Enforcement disabled, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// Do not enforce on StackRox and system namespaces
	if req.Namespace == namespaces.StackRox || kubernetes.SystemNamespaceSet.Contains(req.Namespace) {
		log.Debugf("Action affects system namespace, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// We don't enforce on subresources.
	if req.SubResource != "" {
		log.Debugf("Request is for a subresource, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	if !s.activeForOperation(req.Operation) {
		log.Debugf("Not enforcing on operation, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// Allow the request if it comes from a blessed user
	if s.bypassForUsers.Contains(req.UserInfo.Username) {
		log.Debugf("Request comes from privileged user %s, bypassing %s request on %s/%s [%s]", req.UserInfo.Username, req.Operation, req.Namespace, req.Name, req.Kind)
		return true
	}

	// Allow the request if it comes from a service account in a system namespace
	if strings.HasPrefix(req.UserInfo.Username, "system:serviceaccount:") {
		saNamespace, _ := stringutils.Split2(req.UserInfo.Username[len("system:serviceaccount:"):], ":")
		if kubernetes.SystemNamespaceSet.Contains(saNamespace) {
			log.Debugf("Request comes from a system service account %s, bypassing %s request on %s/%s [%s]", req.UserInfo.Username, req.Operation, req.Namespace, req.Name, req.Kind)
			return true
		}
	}

	// Allow the request if it comes from a blessed group
	for _, group := range req.UserInfo.Groups {
		if s.bypassForGroups.Contains(group) {
			log.Debugf("Request comes from privileged group %s, bypassing %s request on %s/%s [%s]", group, req.Operation, req.Namespace, req.Name, req.Kind)
			return true
		}
	}

	return false
}

func (m *manager) evaluateAdmissionRequest(s *state, req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	log.Tracef("Evaluating request %+v", req)

	if m.shouldBypass(s, req) {
		return pass(req.UID), nil
	}

	log.Debugf("Not bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)

	k8sObj, err := unmarshalK8sObject(req.Kind, req.Object.Raw)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal object from request")
	}

	deployment, err := resources.NewDeploymentFromStaticResource(k8sObj, req.Kind.Kind, s.GetClusterConfig().GetRegistryOverride())
	if err != nil {
		return nil, errors.Wrap(err, "could not convert Kubernetes object into StackRox deployment")
	}

	if deployment == nil {
		log.Debugf("Non-top-level object, bypassing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return pass(req.UID), nil // we only enforce on top-level objects
	}

	log.Debugf("Evaluating policies on %+v", deployment)

	// Check if the deployment has a bypass annotation
	if !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass() {
		if !enforcers.ShouldEnforce(deployment.GetAnnotations()) {
			log.Warnf("deployment %s/%s of type %v was deployed without being checked due to matching bypass annotation %q",
				deployment.GetNamespace(), deployment.GetName(), req.Kind, enforcers.EnforcementBypassAnnotationKey)
			return pass(req.UID), nil
		}
	}

	images := m.getImages(deployment)
	alerts, err := s.detector.Detect(detectionCtx, deployment, images)
	if err != nil {
		return nil, errors.Wrap(err, "running StackRox detection")
	}

	if len(alerts) == 0 {
		log.Debugf("No policies triggered, allowing %s request on %s/%s [%s]", req.Operation, req.Namespace, req.Name, req.Kind)
		return pass(req.UID), nil
	}

	noun := "policies"
	if len(alerts) == 1 {
		noun = "policy"
	}

	// We add a line break at the beginning to look nicer in kubectl
	msgHeader := fmt.Sprintf("\nThe attempted operation violated %d enforced %s, described below:\n\n", len(alerts), noun)
	data := map[string]interface{}{
		"Alerts": alerts,
	}
	if !s.GetClusterConfig().GetAdmissionControllerConfig().GetDisableBypass() {
		data["BypassAnnotationKey"] = enforcers.EnforcementBypassAnnotationKey
	}
	msgBody, err := templates.ExecuteToString(msgTemplate, data)
	if err != nil {
		msgBody = fmt.Sprintf("Internal error executing message template: %v", err)
	}

	return fail(req.UID, msgHeader+msgBody+"\n\n"), nil
}
