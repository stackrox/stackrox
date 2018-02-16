package notifiers

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/readable"
)

type policyFormatStruct struct {
	*v1.Alert

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

	{{if .Policy.ImagePolicy }}{{ subheader "Image Assurance:"}}
		{{if .Policy.ImagePolicy.ImageName}}{{list "Image Name"}}
			{{if .Policy.ImagePolicy.ImageName.Registry}}{{stringify "Registry:" .Policy.ImagePolicy.ImageName.Registry | nestedList}}{{end}}
			{{if .Policy.ImagePolicy.ImageName.Namespace}}{{stringify "Namespace:" .Policy.ImagePolicy.ImageName.Namespace | nestedList}}{{end}}
			{{if .Policy.ImagePolicy.ImageName.Repo}}{{stringify "Repo:" .Policy.ImagePolicy.ImageName.Repo | nestedList}}{{end}}
			{{if .Policy.ImagePolicy.ImageName.Tag}}{{stringify "Tag:" .Policy.ImagePolicy.ImageName.Tag | nestedList}}{{end}}
		{{end}}
		{{if .Policy.ImagePolicy.LineRule}}{{list "Dockerfile Line"}}
			{{if .Policy.ImagePolicy.LineRule.Instruction}}{{stringify "Instruction:" .Policy.ImagePolicy.LineRule.Instruction | nestedList}}{{end}}
			{{if .Policy.ImagePolicy.LineRule.Value}}{{stringify "Value:" .Policy.ImagePolicy.LineRule.Value | nestedList}}{{end}}
		{{end}}
		{{if .Policy.ImagePolicy.SetImageAgeDays}}{{stringify "Image Age >" .Policy.ImagePolicy.GetImageAgeDays "days" | list}}{{end}}
		{{if .Policy.ImagePolicy.Cvss}}{{stringify .CVSS | list}}{{end}}
		{{if .Policy.ImagePolicy.Cve}}{{stringify "CVE:" .Policy.ImagePolicy.Cve | list}}{{end}}
		{{if .Policy.ImagePolicy.Component}}{{list "Component"}}
			{{if .Policy.ImagePolicy.Component.Name}}{{stringify "Name:" .Policy.ImagePolicy.Component.Name | nestedList}}{{end}}
			{{if .Policy.ImagePolicy.Component.Version}}{{stringify "Version:" .Policy.ImagePolicy.Component.Version | nestedList}}{{end}}
		{{end}}
		{{if .Policy.ImagePolicy.SetScanAgeDays}}{{stringify "Scan Age >" .Policy.ImagePolicy.GetScanAgeDays "days" | list}}{{end}}
	{{end}}
	{{if .Policy.PrivilegePolicy }}{{subheader "Privilege Assurance:"}}
		{{if .Policy.PrivilegePolicy.AddCapabilities}}{{list "Disallowed Add-Capabilities"}}
			{{range .Policy.PrivilegePolicy.AddCapabilities}}{{nestedList .}}
			{{end}}
		{{end}}
		{{if .Policy.PrivilegePolicy.DropCapabilities}}{{list "Required Drop-Capabilities"}}
			{{range .Policy.PrivilegePolicy.DropCapabilities}}{{nestedList .}}
			{{end}}
		{{end}}
		{{if .Policy.PrivilegePolicy.Selinux}}{{list "SELinux Security Context"}}
			{{if .Policy.PrivilegePolicy.Selinux.User}}{{stringify "User:" .Policy.PrivilegePolicy.Selinux.User | nestedList}}{{end}}
			{{if .Policy.PrivilegePolicy.Selinux.Role}}{{stringify "Role:" .Policy.PrivilegePolicy.Selinux.Role | nestedList}}{{end}}
			{{if .Policy.PrivilegePolicy.Selinux.Type}}{{stringify "Type:" .Policy.PrivilegePolicy.Selinux.Type | nestedList}}{{end}}
			{{if .Policy.PrivilegePolicy.Selinux.Level}}{{stringify "Level:" .Policy.PrivilegePolicy.Selinux.Level | nestedList}}{{end}}
		{{end}}
		{{if .Policy.PrivilegePolicy.SetPrivileged}}{{stringify "Privileged:" .Policy.PrivilegePolicy.GetPrivileged | list}}{{end}}
	{{end}}
	{{if .Policy.ConfigurationPolicy }}{{subheader "Configuration Assurance:"}}
		{{if .Policy.ConfigurationPolicy.Directory}}{{stringify "Directory:" .Policy.ConfigurationPolicy.Directory | list}}{{end}}
		{{if .Policy.ConfigurationPolicy.Args}}{{stringify "Args:" .Policy.ConfigurationPolicy.Args | list}}{{end}}
		{{if .Policy.ConfigurationPolicy.Command}}{{stringify "Command:" .Policy.ConfigurationPolicy.Command | list}}{{end}}
		{{if .Policy.ConfigurationPolicy.Env}}{{list "Disallowed Environment Variable"}}
			{{if .Policy.ConfigurationPolicy.Env.Key}}{{stringify "Key:" .Policy.ConfigurationPolicy.Env.Key | nestedList}}{{end}}
			{{if .Policy.ConfigurationPolicy.Env.Value}}{{stringify "Value:" .Policy.ConfigurationPolicy.Env.Value | nestedList}}{{end}}
		{{end}}
		{{if .Policy.ConfigurationPolicy.PortPolicy}}{{stringify "Port:" .Port | list}}{{end}}
		{{if .Policy.ConfigurationPolicy.User}}{{stringify "User:" .Policy.ConfigurationPolicy.User | list}}{{end}}
		{{if .Policy.ConfigurationPolicy.VolumePolicy}}{{list "Volume"}}
			{{if .Policy.ConfigurationPolicy.VolumePolicy.Name}}{{stringify "Name:" .Policy.ConfigurationPolicy.VolumePolicy.Name | nestedList}}{{end}}
			{{if .Policy.ConfigurationPolicy.VolumePolicy.Type}}{{stringify "Type:" .Policy.ConfigurationPolicy.VolumePolicy.Type | nestedList}}{{end}}
			{{if .Policy.ConfigurationPolicy.VolumePolicy.Path}}{{stringify "Path:" .Policy.ConfigurationPolicy.VolumePolicy.Path | nestedList}}{{end}}
			{{if .Policy.ConfigurationPolicy.VolumePolicy.SetReadOnly}}{{stringify "ReadOnly:" .Policy.ConfigurationPolicy.VolumePolicy.GetReadOnly | nestedList}}{{end}}
		{{end}}
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
func FormatPolicy(alert *v1.Alert, alertLink string, funcMap template.FuncMap) (string, error) {
	if funcMap == nil {
		return "", fmt.Errorf("Function map passed to FormatPolicy cannot be nil")
	}
	for k := range requiredFunctions {
		if _, ok := funcMap[k]; !ok {
			return "", fmt.Errorf("FuncMap key '%v' must be defined", k)
		}
	}
	funcMap["stringify"] = stringify
	portPolicy := alert.GetPolicy().GetConfigurationPolicy().GetPortPolicy()
	portStr := fmt.Sprintf("%v/%v", portPolicy.GetPort(), portPolicy.GetProtocol())
	data := policyFormatStruct{
		Alert:     alert,
		AlertLink: alertLink,
		CVSS:      readable.NumericalPolicy(alert.GetPolicy().GetImagePolicy().GetCvss(), "cvss"),
		Images:    images.FromContainers(alert.GetDeployment().GetContainers()).String(),
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

type benchmarkFormatStruct struct {
	*v1.BenchmarkSchedule

	Link string
}

const benchmarkFormat = `
New benchmark results for benchmark '{{.BenchmarkSchedule.Name }}' have been posted. Go to {{ .Link }} to view the results.
`

// FormatBenchmark takes in a benchmark, and a link and generates the notification
func FormatBenchmark(schedule *v1.BenchmarkSchedule, scheduleLink string) (string, error) {
	funcMap := make(template.FuncMap)
	funcMap["stringify"] = stringify
	data := benchmarkFormatStruct{
		BenchmarkSchedule: schedule,
		Link:              scheduleLink,
	}
	// Remove all the formatting
	f := strings.Replace(benchmarkFormat, "\t", "", -1)
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

// stringify converts a list of interfaces into a space separated string of their string representations
func stringify(inter ...interface{}) string {
	result := make([]string, len(inter))
	for i, in := range inter {
		result[i] = fmt.Sprintf("%v", in)
	}
	return strings.Join(result, " ")
}
