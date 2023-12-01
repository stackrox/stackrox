package scannerv4

import (
	gogotypes "github.com/gogo/protobuf/types"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// TODO: Add tests when have data to work with

func imageScan(report *v4.VulnerabilityReport) *storage.ImageScan {
	scan := &storage.ImageScan{
		ScanTime:        gogotypes.TimestampNow(),
		Components:      components(report),
		OperatingSystem: os(report),
	}

	if scan.GetOperatingSystem() == "unknown" {
		scan.Notes = append(scan.Notes, storage.ImageScan_OS_UNAVAILABLE)
	}

	return scan
}

func components(report *v4.VulnerabilityReport) []*storage.EmbeddedImageScanComponent {
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(report.PackageVulnerabilities))
	for _, pkg := range report.GetContents().GetPackages() {
		id := pkg.GetId()
		vulnIDs := report.GetPackageVulnerabilities()[id].GetValues()
		component := &storage.EmbeddedImageScanComponent{
			Name:    pkg.GetName(),
			Version: pkg.GetVersion(),
			Vulns:   vulnerabilities(report.Vulnerabilities, vulnIDs),
		}

		components = append(components, component)

	}

	return components
}

func vulnerabilities(vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability, ids []string) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(ids))
	uniqueVulns := set.NewStringSet()
	for _, id := range ids {
		ccVuln := vulnerabilities[id]
		if !uniqueVulns.Add(ccVuln.Name) {
			// Already added this vulnerability, so ignore it.
			continue
		}

		vuln := &storage.EmbeddedVulnerability{
			Cve:               ccVuln.Name,
			Summary:           ccVuln.Description,
			Link:              ccVuln.Link,
			PublishedOn:       ccVuln.GetIssued(),
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			Severity:          normalizedSeverity(ccVuln.GetNormalizedSeverity()),
		}

		if ccVuln.GetFixedInVersion() != "" {
			vuln.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: ccVuln.FixedInVersion,
			}
		}

		vulns = append(vulns, vuln)
	}

	return vulns
}

func normalizedSeverity(severity v4.VulnerabilityReport_Vulnerability_Severity) storage.VulnerabilitySeverity {
	switch severity {
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_LOW:
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE:
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_IMPORTANT:
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL:
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

// os retrieves the OS name:version for the image represented by the given
// vulnerability report.
// If there are zero known distributions for the image or if there are multiple distributions,
// return "unknown", as StackRox only supports a single base-OS at this time.
func os(report *v4.VulnerabilityReport) string {
	if len(report.GetContents().GetDistributions()) == 1 {
		for _, dist := range report.GetContents().GetDistributions() {
			return dist.Did + ":" + dist.VersionId
		}
	}

	return "unknown"
}
