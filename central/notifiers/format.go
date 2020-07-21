package notifiers

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/set"
)

type policyFormatStruct struct {
	*storage.Alert

	AlertLink string
	CVSS      string
	Images    string
	Port      string
	Severity  string
	Time      string
}

const bplPolicyFormat = `
{{stringify "Alert ID:" .Id | line}}
{{stringify "Alert URL:" .AlertLink | line}}
{{stringify "Time (UTC):" .Time | line}}
{{stringify "Severity:" .Severity | line}}
{{header "Violations:"}}
	{{range .Violations}}{{list .Message}}{{end}}
{{header "Policy Definition:"}}
	{{"Description:" | subheader}}
	{{.Policy.Description | list}}
	{{"Rationale:" | subheader}}
	{{.Policy.Rationale | list}}
	{{"Remediation:" | subheader}}
	{{.Policy.Remediation | list}}

	{{ subheader "Policy Criteria:"}}
	{{range .Policy.PolicySections}}
		{{ stringify "Section:" .SectionName | section}}
		{{range .PolicyGroups}}
			{{group .FieldName}}{{": "}}{{valuePrinter .Values .BooleanOperator .Negate}}
		{{end}}
	{{end}}

	{{if .Deployment}}{{line ""}}{{subheader "Deployment:"}}
		{{stringify "ID:" .Deployment.Id | list}}
		{{stringify "Name:" .Deployment.Name | list}}
		{{stringify "Cluster:" .Deployment.ClusterName | list}}
		{{stringify "ClusterId:" .Deployment.ClusterId | list}}
		{{if .Deployment.Namespace }}{{stringify "Namespace:" .Deployment.Namespace | list}}{{end}}
		{{stringify "Images:"  .Images | list}}
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

// FormatPolicy takes in an alert, a link and funcMap that must define specific formatting functions
func FormatPolicy(alert *storage.Alert, alertLink string, funcMap template.FuncMap) (string, error) {
	if funcMap == nil {
		return "", errors.New("Function map passed to FormatPolicy cannot be nil")
	}
	for _, k := range requiredFunctions.AsSlice() {
		if _, ok := funcMap[k]; !ok {
			return "", fmt.Errorf("FuncMap key '%v' must be defined", k)
		}
	}
	funcMap["stringify"] = stringify
	if _, ok := funcMap["valuePrinter"]; !ok {
		funcMap["valuePrinter"] = valuePrinter
	}
	portPolicy := alert.GetPolicy().GetFields().GetPortPolicy()
	portStr := fmt.Sprintf("%v/%v", portPolicy.GetPort(), portPolicy.GetProtocol())
	data := policyFormatStruct{
		Alert:     alert,
		AlertLink: alertLink,
		CVSS:      readable.NumericalPolicy(alert.GetPolicy().GetFields().GetCvss(), "cvss"),
		Images:    types.FromContainers(alert.GetDeployment().GetContainers()).String(),
		Port:      portStr,
		Severity:  SeverityString(alert.Policy.Severity),
		Time:      readable.ProtoTime(alert.Time),
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

// stringify converts a list of interfaces into a space separated string of their string representations
func stringify(inter ...interface{}) string {
	result := make([]string, len(inter))
	for i, in := range inter {
		result[i] = fmt.Sprintf("%v", in)
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

	valuesString = valuesString + "\r\n"

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
