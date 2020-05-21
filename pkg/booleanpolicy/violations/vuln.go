package violations

import (
	"strings"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/search"
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

func cvePrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	// Example message: Fixable CVE-2020-0101 (CVSS 8.2) found in component nginx 1.12.0-debian1ubuntu2 in container “nginx-proxy
	msgTemplate := `
    {{- if .FixedBy}}Fixable {{end}}{{.CVE}}{{if .CVSS}} (CVSS {{.CVSS}}){{end}} found
    {{- if .Component}} in component {{.Component}}-{{.ComponentVersion}}{{end}}
    {{- if .ContainerName }} in container '{{.ContainerName}}'{{end}}
    {{- if .FixedBy}}, resolved by version {{.FixedBy}}{{end}}`
	type cveResultFields struct {
		ContainerName    string
		ImageName        string
		CVE              string
		CVSS             string
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
	r.FixedBy = maybeGetSingleValueFromFieldMap(search.FixedBy.String(), fieldMap)
	r.Component, r.ComponentVersion = getComponentAndVersion(fieldMap)
	return executeTemplate(msgTemplate, r)
}

func componentPrinter(sectionName string, fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		ContainerName    string
		Component        string
		ComponentVersion string
	}
	msgTemplate := "{{if .ContainerName}}Container '{{.ContainerName}}' includes{{else}}Image includes{{end}} component {{.Component}}{{ if .ComponentVersion }} {{.ComponentVersion}}{{end}}"

	r := resultFields{}
	r.ContainerName = maybeGetSingleValueFromFieldMap(augmentedobjs.ContainerNameCustomTag, fieldMap)
	r.Component, r.ComponentVersion = getComponentAndVersion(fieldMap)
	return executeTemplate(msgTemplate, r)
}
