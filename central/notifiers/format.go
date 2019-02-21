package notifiers

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/readable"
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

const policyFormat = `
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

	{{ subheader "Policy Fields:"}}
	{{if .Policy.Fields.ImageName}}{{list "Image Name"}}
		{{if .Policy.Fields.ImageName.Registry}}{{stringify "Registry:" .Policy.Fields.ImageName.Registry | nestedList}}{{end}}
		{{if .Policy.Fields.ImageName.Remote}}{{stringify "Remote:" .Policy.Fields.ImageName.Remote | nestedList}}{{end}}
		{{if .Policy.Fields.ImageName.Tag}}{{stringify "Tag:" .Policy.Fields.ImageName.Tag | nestedList}}{{end}}
	{{end}}
	{{if .Policy.Fields.LineRule}}{{list "Dockerfile Line"}}
		{{if .Policy.Fields.LineRule.Instruction}}{{stringify "Instruction:" .Policy.Fields.LineRule.Instruction | nestedList}}{{end}}
		{{if .Policy.Fields.LineRule.Value}}{{stringify "Value:" .Policy.Fields.LineRule.Value | nestedList}}{{end}}
	{{end}}
	{{if .Policy.Fields.SetImageAgeDays}}{{stringify "Image Age >" .Policy.Fields.GetImageAgeDays "days" | list}}{{end}}
	{{if .Policy.Fields.Cvss}}{{stringify .CVSS | list}}{{end}}
	{{if .Policy.Fields.Cve}}{{stringify "CVE:" .Policy.Fields.Cve | list}}{{end}}
	{{if .Policy.Fields.Component}}{{list "Component"}}
		{{if .Policy.Fields.Component.Name}}{{stringify "Name:" .Policy.Fields.Component.Name | nestedList}}{{end}}
		{{if .Policy.Fields.Component.Version}}{{stringify "Version:" .Policy.Fields.Component.Version | nestedList}}{{end}}
	{{end}}
	{{if .Policy.Fields.SetScanAgeDays}}{{stringify "Scan Age >" .Policy.Fields.GetScanAgeDays "days" | list}}{{end}}
	{{if .Policy.Fields.AddCapabilities}}{{list "Disallowed Add-Capabilities"}}
		{{range .Policy.Fields.AddCapabilities}}{{nestedList .}}
		{{end}}
	{{end}}
	{{if .Policy.Fields.DropCapabilities}}{{list "Required Drop-Capabilities"}}
		{{range .Policy.Fields.DropCapabilities}}{{nestedList .}}
		{{end}}
	{{end}}
	{{if .Policy.Fields.SetPrivileged}}{{stringify "Privileged:" .Policy.Fields.GetPrivileged | list}}{{end}}
	{{if .Policy.Fields.Directory}}{{stringify "Directory:" .Policy.Fields.Directory | list}}{{end}}
	{{if .Policy.Fields.Args}}{{stringify "Args:" .Policy.Fields.Args | list}}{{end}}
	{{if .Policy.Fields.Command}}{{stringify "Command:" .Policy.Fields.Command | list}}{{end}}
	{{if .Policy.Fields.Env}}{{list "Disallowed Environment Variable"}}
		{{if .Policy.Fields.Env.Key}}{{stringify "Key:" .Policy.Fields.Env.Key | nestedList}}{{end}}
		{{if .Policy.Fields.Env.Value}}{{stringify "Value:" .Policy.Fields.Env.Value | nestedList}}{{end}}
	{{end}}
	{{if .Policy.Fields.PortPolicy}}{{stringify "Port:" .Port | list}}{{end}}
	{{if .Policy.Fields.User}}{{stringify "User:" .Policy.Fields.User | list}}{{end}}
	{{if .Policy.Fields.VolumePolicy}}{{list "Volume"}}
		{{if .Policy.Fields.VolumePolicy.Name}}{{stringify "Name:" .Policy.Fields.VolumePolicy.Name | nestedList}}{{end}}
		{{if .Policy.Fields.VolumePolicy.Type}}{{stringify "Type:" .Policy.Fields.VolumePolicy.Type | nestedList}}{{end}}
		{{if .Policy.Fields.VolumePolicy.Source}}{{stringify "Source:" .Policy.Fields.VolumePolicy.Source | nestedList}}{{end}}
		{{if .Policy.Fields.VolumePolicy.Destination}}{{stringify "Destination:" .Policy.Fields.VolumePolicy.Destination | nestedList}}{{end}}
		{{if .Policy.Fields.VolumePolicy.SetReadOnly}}{{stringify "ReadOnly:" .Policy.Fields.VolumePolicy.GetReadOnly | nestedList}}{{end}}
	{{end}}
	{{if .Deployment}}{{subheader "Deployment:"}}
		{{stringify "ID:" .Deployment.Id | list}}
		{{stringify "Name:" .Deployment.Name | list}}
		{{stringify "ClusterId:" .Deployment.ClusterId | list}}
		{{if .Deployment.Namespace }}{{stringify "Namespace:" .Deployment.Namespace | list}}{{end}}
		{{stringify "Images:"  .Images | list}}
	{{end}}
`

var requiredFunctions = map[string]struct{}{
	"header":     {},
	"subheader":  {},
	"line":       {},
	"list":       {},
	"nestedList": {},
}

// FormatPolicy takes in an alert, a link and funcMap that must define specific formatting functions
func FormatPolicy(alert *storage.Alert, alertLink string, funcMap template.FuncMap) (string, error) {
	if funcMap == nil {
		return "", fmt.Errorf("Function map passed to FormatPolicy cannot be nil")
	}
	for k := range requiredFunctions {
		if _, ok := funcMap[k]; !ok {
			return "", fmt.Errorf("FuncMap key '%v' must be defined", k)
		}
	}
	funcMap["stringify"] = stringify
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
	f := strings.Replace(policyFormat, "\t", "", -1)
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
