package vulns

import (
	"math"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/scancomponent"
)

// ComponentScore holds the numerical value of the score with the associated severity rating.
type ComponentScore struct {
	Value    float32
	severity storage.VulnerabilitySeverity
	Severity string
}

// ProcessComponents takes in a slice of components and outputs the min, max, sum CVSS scores as well as the number of CVEs
func ProcessComponents(components []scancomponent.ScanComponent) (min, max ComponentScore, sum float32, num int) {
	min = ComponentScore{
		Value: math.MaxFloat32,
	}
	max = ComponentScore{
		Value: -math.MaxFloat32,
	}
	for _, component := range components {
		cMin, cMax, cSum, cNum := ProcessComponent(component)
		if cNum == 0 {
			continue
		}

		if cMax.Value > max.Value {
			max = cMax
		}
		if cMin.Value < min.Value {
			min = cMin
		}
		sum += cSum
		num += cNum
	}
	return min, max, sum, num
}

// ProcessComponent takes in a single component and outputs the min, max, sum CVSS scores as well as the number of CVEs
func ProcessComponent(component scancomponent.ScanComponent) (min, max ComponentScore, sum float32, numCVEs int) {
	min = ComponentScore{
		Value: math.MaxFloat32,
	}
	max = ComponentScore{
		Value: -math.MaxFloat32,
	}
	for _, vuln := range component.GetVulns() {
		// Exclude vulnerabilities with unknown severity rating.
		if vuln.GetSeverity() == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
			continue
		}
		score := vulnScore(vuln)
		if score > max.Value {
			max.Value = score
			max.severity = vuln.GetSeverity()
		}
		if score < min.Value {
			min.Value = score
			min.severity = vuln.GetSeverity()
		}
		sum += score * score / 10
		numCVEs++
	}
	max.Severity = cvss.FormatSeverity(max.severity)
	min.Severity = cvss.FormatSeverity(min.severity)
	if numCVEs == 0 {
		return ComponentScore{}, ComponentScore{}, 0, 0
	}
	return min, max, sum, numCVEs
}

// vulnScore returns the score of the vulnerability based on severity rating.
// Previously, we used the CVSS score as the score; however,
// it has since become clear severity rating and score do not always correspond.
// Severity rating is more meaningful when it comes to prioritizing vulnerabilities,
// so we use the severity instead.
// For example, given a vulnerability with CVSSv3.1 score of 9.0 and severity rating Low,
// this will return 2.0 as the score.
func vulnScore(vuln cvss.VulnI) float32 {
	severity := cvss.VulnToSeverity(vuln)

	if vuln.GetScoreVersion() == storage.CVEInfo_V2 {
		return cvss2Score(severity)
	}
	return cvss3Score(severity)
}

// cvss2Score maps storage.VulnerabilitySeverity to a CVSSv2.0 score.
// The score is influenced by https://nvd.nist.gov/vuln-metrics/cvss.
func cvss2Score(severity storage.VulnerabilitySeverity) float32 {
	switch severity {
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return 2.0
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return 5.5
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return 8.5
	default:
		return 0.0
	}
}

// cvss3Score maps storage.VulnerabilitySeverity to a CVSSv3.x score.
// The score is influenced by https://nvd.nist.gov/vuln-metrics/cvss.
func cvss3Score(severity storage.VulnerabilitySeverity) float32 {
	switch severity {
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return 2.0
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return 5.5
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return 8.0
	case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
		return 9.5
	default:
		return 0.0
	}
}
