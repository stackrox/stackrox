package scan

import (
	"sort"
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

type cveJSONStructure struct {
	Summary         map[string]int         `json:"summary"`
	Vulnerabilities []cveVulnerabilityJSON `json:"vulnerabilities,omitempty"`
}

type cveVulnerabilityJSON struct {
	CveID                 string `json:"cveId"`
	CveSeverity           string `json:"cveSeverity"`
	CveInfo               string `json:"cveInfo"`
	ComponentName         string `json:"componentName"`
	ComponentVersion      string `json:"componentVersion"`
	ComponentFixedVersion string `json:"componentFixedVersion"`
}

// newCVESummaryForPrinting creates a cveJSONResult that shall be used for printing and holds
// all relevant information regarding components and CVEs and a summary of all found CVE by
// severity
// NOTE: The returned *cveJSONResult CAN be passed to json.Marshal
func newCVESummaryForPrinting(scanResults *storage.ImageScan) *cveJSONResult {
	var vulnerabilitiesJSON []cveVulnerabilityJSON
	components := sortComponentsByName(scanResults.GetComponents())
	vulnSummaryMap := createNumOfVulnerabilitiesBySeverityMap()
	uniqueCVEs := set.NewStringSet()

	for _, comp := range components {
		vulns := comp.GetVulns()
		vulnsJSON := getVulnerabilityJSON(vulns, comp, vulnSummaryMap, uniqueCVEs)
		if len(vulnsJSON) != 0 {
			vulnerabilitiesJSON = append(vulnerabilitiesJSON, vulnsJSON...)
			vulnSummaryMap[totalComponentsMapKey]++
		}
	}

	return &cveJSONResult{
		Result: cveJSONStructure{
			Summary:         vulnSummaryMap,
			Vulnerabilities: vulnerabilitiesJSON,
		},
	}
}

func getVulnerabilityJSON(vulnerabilities []*storage.EmbeddedVulnerability, comp *storage.EmbeddedImageScanComponent,
	numOfVulnsBySeverity map[string]int, uniqueCVEs set.StringSet) []cveVulnerabilityJSON {
	// sort vulnerabilities by severity
	vulnerabilities = sortVulnerabilitiesForSeverity(vulnerabilities)

	vulnerabilitiesJSON := make([]cveVulnerabilityJSON, 0, len(vulnerabilities))
	for _, vulnerability := range vulnerabilities {
		severity := cveSeverityFromVulnerabilitySeverity(vulnerability.GetSeverity())
		vulnJSON := cveVulnerabilityJSON{
			CveID:                 vulnerability.GetCve(),
			CveSeverity:           severity.String(),
			CveInfo:               vulnerability.GetLink(),
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

func sortVulnerabilitiesForSeverity(vulns []*storage.EmbeddedVulnerability) []*storage.EmbeddedVulnerability {
	sort.SliceStable(vulns, func(p, q int) bool { return vulns[p].GetSeverity() > vulns[q].GetSeverity() })
	return vulns
}

func sortComponentsByName(components []*storage.EmbeddedImageScanComponent) []*storage.EmbeddedImageScanComponent {
	sort.SliceStable(components, func(i, j int) bool {
		return components[i].Name < components[j].Name
	})
	return components
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
