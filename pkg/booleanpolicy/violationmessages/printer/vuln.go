package printer

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/readable"
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

const (
	// Example message: Fixable CVE-2020-0101 (CVSS 8.2, NVD CVSS 8.0, EPSS 75%) (severity Important) found in component 'nginx' (version 1.12.0-debian1ubuntu2) in container 'nginx-proxy', resolved by version 1.13.0-debian0ubuntu1
	// example both nvd cvss and cvss : CVE-2014-0160 (CVSS 6, NVD CVSS 8) (severity Unknown) found in component 'heartbleed' (version 1.2) in container 'nginx'
	cveTemplate = `
    {{- if .FixedBy}}Fixable {{end}}{{.CVE}}
    {{- if .Scores}} ({{.Scores}}){{end}}{{if .Severity}} (severity {{.Severity}}){{end}} found
    {{- if .Component}} in component '{{.Component}}' (version {{.ComponentVersion}}){{end}}
    {{- if .ContainerName }} in container '{{.ContainerName}}'{{end}}
    {{- if .FixedBy}}, resolved by version {{.FixedBy}}{{end}}`
)

func cvePrinter(fieldMap map[string][]string) ([]string, error) {

	type cveResultFields struct {
		ContainerName    string
		ImageName        string
		CVE              string
		Scores           string
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
	var epss string
	if rawEPSS := maybeGetSingleValueFromFieldMap(search.EPSSProbablity.String(), fieldMap); rawEPSS != "" {
		epss = formatEPSSPercent(rawEPSS)
	}
	r.Scores = cveScores(
		maybeGetSingleValueFromFieldMap(search.CVSS.String(), fieldMap),
		maybeGetSingleValueFromFieldMap(search.NVDCVSS.String(), fieldMap),
		epss,
	)
	r.Severity = strings.Title(strings.TrimSuffix(strings.ToLower(maybeGetSingleValueFromFieldMap(search.Severity.String(), fieldMap)), "_vulnerability_severity"))
	r.FixedBy = maybeGetSingleValueFromFieldMap(search.FixedBy.String(), fieldMap)
	r.Component, r.ComponentVersion = getComponentAndVersion(fieldMap)
	return executeTemplate(cveTemplate, r)
}

func cveScores(cvss, nvdCVSS, epss string) string {
	var scores []string
	if cvss != "" {
		scores = append(scores, "CVSS "+cvss)
	}
	if nvdCVSS != "" {
		scores = append(scores, "NVD CVSS "+nvdCVSS)
	}
	if epss != "" {
		scores = append(scores, "EPSS "+epss)
	}
	return strings.Join(scores, ", ")
}

// formatEPSSPercent renders a stored probability (0-1) as a percent for violation text.
func formatEPSSPercent(probability string) string {
	f, err := strconv.ParseFloat(strings.TrimSpace(probability), 64)
	if err != nil {
		return probability
	}
	return readable.Float(f*100, 3) + "%"
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
