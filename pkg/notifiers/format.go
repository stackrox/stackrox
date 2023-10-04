package notifiers

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mitre/datastore"
	mitreUtils "github.com/stackrox/rox/pkg/mitre/utils"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
)

type policyFormatStruct struct {
	*storage.Alert

	FullMitreAttackVectors []*storage.MitreAttackVector

	AlertLink string
	Severity  string
	Time      string

	DeploymentCommaSeparatedImages string

	// Populated only if the entity is an image.
	Image string
}

const bplPolicyFormat = `
{{stringify "Alert ID:" .Id | line}}
{{stringify "Alert URL:" .AlertLink | line}}
{{stringify "Time (UTC):" .Time | line}}
{{stringify "Severity:" .Severity | line}}
{{header "Violations:"}}
	{{range .Violations}}
		{{list .Message}}
		{{if .MessageAttributes}}
			{{if isViolationKeyValue .MessageAttributes }}
				{{if .MessageAttributes.KeyValueAttrs}}
					{{range .MessageAttributes.KeyValueAttrs.Attrs}}
						{{stringify .Key ":" .Value | nestedList}}
					{{end}}
				{{end}}
			{{end}}
		{{end}}
	{{end}}
	{{if .ProcessViolation}}
		{{list .ProcessViolation.Message}}
	{{end}}
{{header "Policy Definition:"}}
	{{"Description:" | subheader}}
	{{.Policy.Description | list}}
	{{"Rationale:" | subheader}}
	{{.Policy.Rationale | list}}
	{{"Remediation:" | subheader}}
	{{.Policy.Remediation | list}}

	{{if .FullMitreAttackVectors}}{{subheader "MITRE ATT&CK:"}}
		{{range .FullMitreAttackVectors}}
			{{stringify "Tactic:" .Tactic.Name "(" .Tactic.Id ")"  | list}}
			{{if .Techniques}}{{nestedList "Techniques:"}}
				{{range .Techniques}}
					{{stringify "\t"}}{{stringify .Name "(" .Id ")" | nestedList}}
				{{end}}
			{{end}}
		{{end}}
	{{end}}

	{{ subheader "Policy Criteria:"}}
	{{range .Policy.PolicySections}}
		{{ stringify "Section" (default .SectionName "Unnamed") ":" | section}}
		{{range .PolicyGroups}}
			{{group .FieldName}}{{": "}}{{valuePrinter .Values .BooleanOperator .Negate}}
		{{end}}
	{{end}}

{{if .GetDeployment}}{{line ""}}{{header "Deployment:"}}
	{{stringify "ID:" .GetDeployment.Id | list}}
	{{stringify "Name:" .GetDeployment.Name | list}}
	{{stringify "Cluster:" .GetDeployment.ClusterName | list}}
	{{stringify "ClusterId:" .GetDeployment.ClusterId | list}}
	{{if .GetDeployment.Namespace }}{{stringify "Namespace:" .GetDeployment.Namespace | list}}{{end}}
	{{stringify "Images:" .DeploymentCommaSeparatedImages | list}}
{{end}}

{{if .GetResource}}{{line ""}}{{header "Resource:"}}
	{{stringify "Name:" .GetResource.Name | list}}
	{{stringify "Type:" .GetResource.ResourceType | list}}
	{{stringify "Cluster:" .GetResource.ClusterName | list}}
	{{stringify "ClusterId:" .GetResource.ClusterId | list}}
	{{if .GetResource.Namespace }}{{stringify "Namespace:" .GetResource.Namespace | list}}{{end}}
{{end}}

{{if .GetImage}}{{line ""}}{{header "Image:"}}
	{{stringify "Name:" .Image | list}}
{{end}}
`

var requiredFunctions = set.NewFrozenStringSet(
	"header",
	"subheader",
	"line",
	"list",
	"nestedList",
	"section",
	"group",
)

// FormatAlert takes in an alert, a link and funcMap that must define specific formatting functions
func FormatAlert(alert *storage.Alert, alertLink string, funcMap template.FuncMap, mitreStore datastore.AttackReadOnlyDataStore) (string, error) {
	if funcMap == nil {
		return "", errors.New("Function map passed to FormatAlert cannot be nil")
	}
	for _, k := range requiredFunctions.AsSlice() {
		if _, ok := funcMap[k]; !ok {
			return "", fmt.Errorf("FuncMap key '%v' must be defined", k)
		}
	}
	funcMap["stringify"] = stringify
	funcMap["default"] = stringutils.OrDefault
	funcMap["isViolationKeyValue"] = isViolationKeyValue
	if _, ok := funcMap["valuePrinter"]; !ok {
		funcMap["valuePrinter"] = valuePrinter
	}

	fullMitreVectors, err := mitreUtils.GetFullMitreAttackVectors(mitreStore, alert.GetPolicy())
	if err != nil {
		log.Errorw("Could not get MITRE details for alert", logging.AlertID(alert.GetId()), logging.Err(err))
	}

	data := policyFormatStruct{
		Alert:                  alert,
		FullMitreAttackVectors: fullMitreVectors,
		AlertLink:              alertLink,
		Severity:               SeverityString(alert.Policy.Severity),
		Time:                   readable.ProtoTime(alert.Time),
	}
	switch alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		data.DeploymentCommaSeparatedImages = types.FromContainers(alert.GetDeployment().GetContainers()).String()
	case *storage.Alert_Image:
		data.Image = types.Wrapper{GenericImage: alert.GetImage()}.FullName()
	}

	// Remove all the formatting
	format := bplPolicyFormat
	f := strings.Replace(format, "\t", "", -1)
	f = strings.Replace(f, "\n", "", -1)

	tmpl, err := template.New("").Funcs(funcMap).Parse(f)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

// SummaryForAlert returns a summary for an alert.
// This can be used for notifiers that need a summary/title for the notification.
func SummaryForAlert(alert *storage.Alert) string {
	switch entity := alert.GetEntity().(type) {
	case *storage.Alert_Deployment_:
		return fmt.Sprintf("Deployment %s (in cluster %s) violates '%s' Policy", entity.Deployment.GetName(), entity.Deployment.GetClusterName(), alert.GetPolicy().GetName())
	case *storage.Alert_Image:
		return fmt.Sprintf("Image %s violates '%s' Policy", types.Wrapper{GenericImage: entity.Image}.FullName(), alert.GetPolicy().GetName())
	case *storage.Alert_Resource_:
		return fmt.Sprintf("Policy '%s' violated in cluster %s", alert.GetPolicy().GetName(), alert.GetResource().GetClusterName())
	}
	return fmt.Sprintf("Policy '%s' violated", alert.GetPolicy().GetName())
}

type networkPolicyFormatStruct struct {
	YAML        string
	ClusterName string
}

const networkPolicyYAMLNotificationFormat = `
	Please review the following network policy YAML that needs to be applied to cluster '{{.ClusterName}}'.
	{{codeBlock .YAML}}
	`

// FormatNetworkPolicyYAML takes in a cluster name and network policy yaml to generate the notification
func FormatNetworkPolicyYAML(yaml string, clusterName string, funcMap template.FuncMap) (string, error) {
	data := networkPolicyFormatStruct{
		YAML:        yaml,
		ClusterName: clusterName,
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(networkPolicyYAMLNotificationFormat)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	err = tmpl.Execute(&tpl, data)
	if err != nil {
		return "", err
	}
	return tpl.String(), nil
}

// isViolationKeyValue returns try if src is of type **storage.Alert_Violation_KeyValueAttrs_
// Used to validate if a one-of is of type KeyValueAttrs within a template
func isViolationKeyValue(src interface{}) bool {
	return reflect.TypeOf(src) == reflect.TypeOf((*storage.Alert_Violation_KeyValueAttrs_)(nil))
}

// stringify converts a list of interfaces into a space separated string of their string representations
func stringify(inter ...interface{}) string {
	result := make([]string, 0, len(inter))
	for _, in := range inter {
		str := fmt.Sprintf("%v", in)
		if str != "" {
			result = append(result, fmt.Sprintf("%v", in))
		}
	}
	return strings.Join(result, " ")
}

func valuePrinter(values []*storage.PolicyValue, op storage.BooleanOperator, negated bool) string {
	var opString string
	if op == storage.BooleanOperator_OR {
		opString = " OR "
	} else {
		opString = " AND "
	}

	var valueStrings []string
	for _, value := range values {
		valueStrings = append(valueStrings, value.GetValue())
	}

	valuesString := strings.Join(valueStrings, opString)
	if negated {
		valuesString = fmt.Sprintf("NOT (%s)", valuesString)
	}

	return valuesString
}

// GetNotifiersCompatiblePolicySeverity converts the enum value to more meaningful policy severity string
func GetNotifiersCompatiblePolicySeverity(enumSeverity string) (string, error) {
	strs := strings.Split(enumSeverity, "_")
	if len(strs) != 2 || strs[1] != "SEVERITY" {
		return "", fmt.Errorf("severity enum %q does not the format *_SEVERITY", enumSeverity)
	}
	return fmt.Sprintf("%s%s", strings.ToUpper(strs[0][:1]), strings.ToLower(strs[0][1:])), nil
}
