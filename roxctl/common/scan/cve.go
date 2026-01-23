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

const (
	LowCVESeverity CVESeverity = iota
	ModerateCVESeverity
	ImportantCVESeverity
	CriticalCVESeverity
)

type CVESeverity int

func (c CVESeverity) String() string {
	return [...]string{"LOW", "MODERATE", "IMPORTANT", "CRITICAL"}[c]
}

func (c CVESeverity) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.String()) //nolint:wrapcheck
}

func cveSeverityFromString(s string) CVESeverity {
	switch s {
	case LowCVESeverity.String():
		return LowCVESeverity
	case ModerateCVESeverity.String():
		return ModerateCVESeverity
	case ImportantCVESeverity.String():
		return ImportantCVESeverity
	case CriticalCVESeverity.String():
		return CriticalCVESeverity
	default:
		return LowCVESeverity
	}
}

// AllSeverities returns the supported CVE severity labels in order
// from lowest to highest.
func AllSeverities() []string {
	return []string{
		LowCVESeverity.String(),
		ModerateCVESeverity.String(),
		ImportantCVESeverity.String(),
		CriticalCVESeverity.String(),
	}
}

type CVEJSONResult struct {
	Result CVEJSONStructure `json:"result"`
}

func (c *CVEJSONResult) CountVulnerabilities() int {
	return c.Result.Summary[totalVulnerabilitiesMapKey]
}

func (c *CVEJSONResult) CountComponents() int {
	return c.Result.Summary[totalComponentsMapKey]
}

type CVEJSONStructure struct {
	Summary         map[string]int         `json:"summary"`
	Vulnerabilities []CVEVulnerabilityJSON `json:"vulnerabilities,omitempty"`
}

type CVEVulnerabilityJSON struct {
	CveID                 string      `json:"cveId"`
	CveSeverity           CVESeverity `json:"cveSeverity"`
	CveCVSS               float32     `json:"cveCVSS"`
	CveInfo               string      `json:"cveInfo"`
	AdvisoryID            string      `json:"advisoryId"`
	AdvisoryInfo          string      `json:"advisoryInfo"`
	ComponentName         string      `json:"componentName"`
	ComponentVersion      string      `json:"componentVersion"`
	ComponentFixedVersion string      `json:"componentFixedVersion"`
}

// NewCVESummaryForPrinting creates a cveJSONResult that shall be used for printing and holds
// all relevant information regarding components and CVEs and a summary of all found CVE by
// severity
// NOTE: The returned *cveJSONResult CAN be passed to json.Marshal
func NewCVESummaryForPrinting(scanResults *storage.ImageScan, severities []string) *CVEJSONResult {
	var vulnerabilitiesJSON []CVEVulnerabilityJSON
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

	return &CVEJSONResult{
		Result: CVEJSONStructure{
			Summary:         vulnSummaryMap,
			Vulnerabilities: vulnerabilitiesJSON,
		},
	}
}

func getVulnerabilityJSON(vulnerabilities []*storage.EmbeddedVulnerability, comp *storage.EmbeddedImageScanComponent,
	numOfVulnsBySeverity map[string]int, uniqueCVEs set.StringSet,
	severitiesToInclude []CVESeverity) []CVEVulnerabilityJSON {
	// sort vulnerabilities by severity and CVSS score
	sortVulnerabilities(vulnerabilities)

	vulnerabilitiesJSON := make([]CVEVulnerabilityJSON, 0, len(vulnerabilities))
	for _, vulnerability := range vulnerabilities {
		severity := cveSeverityFromVulnerabilitySeverity(vulnerability.GetSeverity())
		if !slices.Contains(severitiesToInclude, severity) {
			continue
		}

		vulnJSON := CVEVulnerabilityJSON{
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
func sortVulnerabilityJSONs(vulns []CVEVulnerabilityJSON) {
	componentMaxSeverity := map[string]CVESeverity{}
	for _, v := range vulns {
		componentMaxSeverity[v.ComponentName] = max(v.CveSeverity, componentMaxSeverity[v.ComponentName])
	}
	slices.SortStableFunc(vulns, func(a, b CVEVulnerabilityJSON) int {
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

func cveSeverityFromVulnerabilitySeverity(severity storage.VulnerabilitySeverity) CVESeverity {
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

func createSeveritiesToInclude(severities []string) []CVESeverity {
	severitiesToInclude := make([]CVESeverity, 0, len(severities))
	for _, severity := range severities {
		severitiesToInclude = append(severitiesToInclude, cveSeverityFromString(severity))
	}
	return severitiesToInclude
}
