package scannerv4

import (
	"strings"

	gogotypes "github.com/gogo/protobuf/types"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/set"
)

func imageScan(metadata *storage.ImageMetadata, report *v4.VulnerabilityReport) *storage.ImageScan {
	scan := &storage.ImageScan{
		// ScannerVersion: ,
		ScanTime:        gogotypes.TimestampNow(),
		OperatingSystem: os(report),
		Components:      components(metadata, report),
	}

	if scan.GetOperatingSystem() == "unknown" {
		scan.Notes = append(scan.Notes, storage.ImageScan_OS_UNAVAILABLE)
	}

	return scan
}

func components(metadata *storage.ImageMetadata, report *v4.VulnerabilityReport) []*storage.EmbeddedImageScanComponent {
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(report.GetPackageVulnerabilities()))
	for _, pkg := range report.GetContents().GetPackages() {
		id := pkg.GetId()
		vulnIDs := report.GetPackageVulnerabilities()[id].GetValues()
		component := &storage.EmbeddedImageScanComponent{
			Name:          pkg.GetName(),
			Version:       pkg.GetVersion(),
			Vulns:         vulnerabilities(report.GetVulnerabilities(), vulnIDs),
			Location:      pkg.GetPackageDb(),
			HasLayerIndex: layerIndex(metadata, report, id),
		}

		components = append(components, component)
	}

	return components
}

func layerIndex(metadata *storage.ImageMetadata, report *v4.VulnerabilityReport, pkgID string) *storage.EmbeddedImageScanComponent_LayerIndex {
	layerSHAToIndex := clair.BuildSHAToIndexMap(metadata)

	envList := report.GetContents().GetEnvironments()[pkgID]
	if len(envList.GetEnvironments()) > 0 {
		env := envList.GetEnvironments()[0]

		if val, ok := layerSHAToIndex[env.GetIntroducedIn()]; ok {
			return &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: val,
			}
		}
	}

	return nil
}

func vulnerabilities(vulnerabilities map[string]*v4.VulnerabilityReport_Vulnerability, ids []string) []*storage.EmbeddedVulnerability {
	if len(vulnerabilities) == 0 {
		return nil
	}

	vulns := make([]*storage.EmbeddedVulnerability, 0, len(ids))
	uniqueVulns := set.NewStringSet()
	for _, id := range ids {
		ccVuln := vulnerabilities[id]
		if !uniqueVulns.Add(ccVuln.Name) {
			// Already added this vulnerability, so ignore it.
			continue
		}

		vuln := &storage.EmbeddedVulnerability{
			Cve: ccVuln.GetName(),
			// Cvss: ,
			Summary: ccVuln.GetDescription(),
			Link:    link(ccVuln.GetLink()),
			// ScoreVersion: ,
			// CvssV2: ,
			// CvssV3: ,
			PublishedOn: ccVuln.GetIssued(),
			// LastModified: ,
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			Severity:          normalizedSeverity(ccVuln.GetNormalizedSeverity()),
		}

		if ccVuln.GetFixedInVersion() != "" {
			vuln.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: ccVuln.GetFixedInVersion(),
			}
		}

		vulns = append(vulns, vuln)
	}

	return vulns
}

// link returns the first link from space separated list of links (which is how ClairCore provides links).
// The ACS UI will fail to show a vulnerability's link if it is an invalid URL.
func link(multipleLinks string) string {
	link := multipleLinks
	if links := strings.Split(link, " "); len(links) > 1 {
		link = links[0]
	}

	return link
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
