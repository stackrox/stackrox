package printer

import (
	"strings"

	"github.com/stackrox/stackrox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/stackrox/pkg/search"
)

func getComponentAndVersion(fieldMap map[string][]string) (component string, version string) {
	if value, err := getSingleValueFromFieldMap(augmentedobjs.ComponentAndVersionCustomTag, fieldMap); err == nil {
		componentAndVersion := strings.SplitN(value, augmentedobjs.CompositeFieldCharSep, 2)
		component = componentAndVersion[0]
		if len(componentAndVersion) > 1 {
			version = componentAndVersion[1]
		}
	}
	return component, version
}

const (
	// Example message: Fixable CVE-2020-0101 (CVSS 8.2) (severity Important) found in component 'nginx' (version 1.12.0-debian1ubuntu2) in container 'nginx-proxy', resolved by version 1.13.0-debian0ubuntu1
	cveTemplate = `
    {{- if .FixedBy}}Fixable {{end}}{{.CVE}}{{if .CVSS}} (CVSS {{.CVSS}}){{end}}{{if .Severity}} (severity {{.Severity}}){{end}} found
    {{- if .Component}} in component '{{.Component}}' (version {{.ComponentVersion}}){{end}}
    {{- if .ContainerName }} in container '{{.ContainerName}}'{{end}}
    {{- if .FixedBy}}, resolved by version {{.FixedBy}}{{end}}`
)

func cvePrinter(fieldMap map[string][]string) ([]string, error) {

	type cveResultFields struct {
		ContainerName    string
		ImageName        string
		CVE              string
		CVSS             string
		Severity         string
		FixedBy          string
		Component        string
		ComponentVersion string
	}
	r := cveResultFields{}

	var err error
	if r.CVE, err = getSingleValueFromFieldMap(search.CVE.String(), fieldMap); err != nil {
		return nil, err
	}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	r.CVSS = maybeGetSingleValueFromFieldMap(search.CVSS.String(), fieldMap)
	r.Severity = strings.Title(strings.TrimSuffix(strings.ToLower(maybeGetSingleValueFromFieldMap(search.Severity.String(), fieldMap)), "_vulnerability_severity"))
	r.FixedBy = maybeGetSingleValueFromFieldMap(search.FixedBy.String(), fieldMap)
	r.Component, r.ComponentVersion = getComponentAndVersion(fieldMap)
	return executeTemplate(cveTemplate, r)
}

const (
	componentTemplate = `{{if .ContainerName}}Container '{{.ContainerName}}' includes{{else}}Image includes{{end}} component '{{.Component}}'{{ if .ComponentVersion }} (version {{.ComponentVersion}}){{end}}`
)

func componentPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName    string
		Component        string
		ComponentVersion string
	}

	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	r.Component, r.ComponentVersion = getComponentAndVersion(fieldMap)
	return executeTemplate(componentTemplate, r)
}
