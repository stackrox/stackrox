package scan

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

const (
	totalVulnerabilitiesMapKey = "TOTAL-VULNERABILITIES"
	totalComponentsMapKey      = "TOTAL-COMPONENTS"
)

type cveSeverity int

const (
	lowCVESeverity cveSeverity = iota
	moderateCVESeverity
	importantCVESeverity
	criticalCVESeverity
)

func (c cveSeverity) String() string {
	return [...]string{"LOW", "MODERATE", "IMPORTANT", "CRITICAL"}[c]
}

func (c cveSeverity) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String()) //nolint:wrapcheck
}

func cveSeverityFromString(s string) cveSeverity {
	switch s {
	case lowCVESeverity.String():
		return lowCVESeverity
	case moderateCVESeverity.String():
		return moderateCVESeverity
	case importantCVESeverity.String():
		return importantCVESeverity
	case criticalCVESeverity.String():
		return criticalCVESeverity
	default:
		return lowCVESeverity
	}
}

type cveJSONResult struct {
	Result cveJSONStructure `json:"result"`
}

func (c *cveJSONResult) CountVulnerabilities() int {
	return c.Result.Summary[totalVulnerabilitiesMapKey]
}

func (c *cveJSONResult) CountComponents() int {
	return c.Result.Summary[totalComponentsMapKey]
}

type cveJSONStructure struct {
	Summary         map[string]int         `json:"summary"`
	Vulnerabilities []cveVulnerabilityJSON `json:"vulnerabilities,omitempty"`
}

type cveVulnerabilityJSON struct {
	CveID                 string      `json:"cveId"`
	CveSeverity           cveSeverity `json:"cveSeverity"`
	CveCVSS               float32     `json:"cveCVSS"`
	CveInfo               string      `json:"cveInfo"`
	AdvisoryID            string      `json:"advisoryId"`
	AdvisoryInfo          string      `json:"advisoryInfo"`
	ComponentName         string      `json:"componentName"`
	ComponentVersion      string      `json:"componentVersion"`
	ComponentFixedVersion string      `json:"componentFixedVersion"`
}

// newCVESummaryForPrinting creates a cveJSONResult that shall be used for printing and holds
// all relevant information regarding components and CVEs and a summary of all found CVE by
// severity
// NOTE: The returned *cveJSONResult CAN be passed to json.Marshal
func newCVESummaryForPrinting(scanResults *storage.ImageScan, severities []string) *cveJSONResult {
	var vulnerabilitiesJSON []cveVulnerabilityJSON
	vulnSummaryMap := createNumOfVulnerabilitiesBySeverityMap()
	severitiesToInclude := createSeveritiesToInclude(severities)
	uniqueCVEs := set.NewStringSet()

	for _, comp := range scanResults.GetComponents() {
		vulns := comp.GetVulns()
		vulnsJSON := getVulnerabilityJSON(vulns, comp, vulnSummaryMap, uniqueCVEs, severitiesToInclude)
		if len(vulnsJSON) != 0 {
			vulnerabilitiesJSON = append(vulnerabilitiesJSON, vulnsJSON...)
			vulnSummaryMap[totalComponentsMapKey]++
		}
	}

	sortVulnerabilityJSONs(vulnerabilitiesJSON)

	return &cveJSONResult{
		Result: cveJSONStructure{
			Summary:         vulnSummaryMap,
			Vulnerabilities: vulnerabilitiesJSON,
		},
	}
}

func getVulnerabilityJSON(vulnerabilities []*storage.EmbeddedVulnerability, comp *storage.EmbeddedImageScanComponent,
	numOfVulnsBySeverity map[string]int, uniqueCVEs set.StringSet,
	severitiesToInclude []cveSeverity) []cveVulnerabilityJSON {
	// sort vulnerabilities by severity and CVSS score
	sortVulnerabilities(vulnerabilities)

	vulnerabilitiesJSON := make([]cveVulnerabilityJSON, 0, len(vulnerabilities))
	for _, vulnerability := range vulnerabilities {
		severity := cveSeverityFromVulnerabilitySeverity(vulnerability.GetSeverity())
		if !slices.Contains(severitiesToInclude, severity) {
			continue
		}

		vulnJSON := cveVulnerabilityJSON{
			CveID:                 vulnerability.GetCve(),
			CveSeverity:           severity,
			CveCVSS:               vulnerability.GetCvss(),
			CveInfo:               vulnerability.GetLink(),
			AdvisoryID:            vulnerability.GetAdvisory().GetName(),
			AdvisoryInfo:          vulnerability.GetAdvisory().GetLink(),
			ComponentName:         comp.GetName(),
			ComponentVersion:      comp.GetVersion(),
			ComponentFixedVersion: vulnerability.GetFixedBy(),
		}

		// Only increase the number of vulnerabilities by severity if the CVE ID has not been added before.
		// The severity will also match across CVEs, since they are assigned to the unique CVE.
		if uniqueCVEs.Add(vulnerability.GetCve()) {
			numOfVulnsBySeverity[totalVulnerabilitiesMapKey] = uniqueCVEs.Cardinality()
			numOfVulnsBySeverity[severity.String()]++
		}
		vulnerabilitiesJSON = append(vulnerabilitiesJSON, vulnJSON)
	}
	return vulnerabilitiesJSON
}

// sortVulnerabilityJSONs sorts the given slice of vulnerabilityJSON structs in decreasing order by the component's maximum severity.
// If two severities match, then they are sorted in increasing order by component name.
// If the names also match, then they are sorted in decreasing order by severity.
// If the severities also match, then they are sorted in decreasing order by CVSS score.
func sortVulnerabilityJSONs(vulns []cveVulnerabilityJSON) {
	componentMaxSeverity := map[string]cveSeverity{}
	for _, v := range vulns {
		componentMaxSeverity[v.ComponentName] = max(v.CveSeverity, componentMaxSeverity[v.ComponentName])
	}
	slices.SortStableFunc(vulns, func(a, b cveVulnerabilityJSON) int {
		sevAMax, sevBMax := componentMaxSeverity[a.ComponentName], componentMaxSeverity[b.ComponentName]
		if c := cmp.Compare(sevBMax, sevAMax); c != 0 {
			return c
		}
		if c := cmp.Compare(a.ComponentName, b.ComponentName); c != 0 {
			return c
		}
		if c := cmp.Compare(b.CveSeverity, a.CveSeverity); c != 0 {
			return c
		}
		return cmp.Compare(b.CveCVSS, a.CveCVSS)
	})
}

// sortVulnerabilities sorts the given slice of vulnerabilities in decreasing order by severity.
// If two severities match, then they are sorted in decreasing order by CVSS score.
func sortVulnerabilities(vulns []*storage.EmbeddedVulnerability) {
	slices.SortStableFunc(vulns, func(a, b *storage.EmbeddedVulnerability) int {
		if c := cmp.Compare(b.GetSeverity(), a.GetSeverity()); c != 0 {
			return c
		}
		return cmp.Compare(b.GetCvss(), a.GetCvss())
	})
}

func stripVulnerabilitySevEnum(severityName string) string {
	// replace "LOW_VULNERABILITY_SEVERITY" to "LOW"
	return strings.TrimSuffix(severityName, "_VULNERABILITY_SEVERITY")
}

func cveSeverityFromVulnerabilitySeverity(severity storage.VulnerabilitySeverity) cveSeverity {
	return cveSeverityFromString(stripVulnerabilitySevEnum(severity.String()))
}

func createNumOfVulnerabilitiesBySeverityMap() map[string]int {
	var m = map[string]int{}
	for _, enum := range storage.VulnerabilitySeverity_name {
		// skip the "UNKNOWN" vulnerability
		if enum == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY.String() {
			continue
		}
		m[stripVulnerabilitySevEnum(enum)] = 0
	}
	m[totalVulnerabilitiesMapKey] = 0
	m[totalComponentsMapKey] = 0
	return m
}

func createSeveritiesToInclude(severities []string) []cveSeverity {
	severitiesToInclude := make([]cveSeverity, 0, len(severities))
	for _, severity := range severities {
		severitiesToInclude = append(severitiesToInclude, cveSeverityFromString(severity))
	}
	return severitiesToInclude
}
