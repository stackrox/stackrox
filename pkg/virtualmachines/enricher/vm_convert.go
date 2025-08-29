package enricher

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
)

func toVirtualMachineScan(vr *v4.VulnerabilityReport) *storage.VirtualMachineScan {
	return &storage.VirtualMachineScan{
		ScanTime:   protocompat.TimestampNow(),
		Components: toImageScanComponents(vr),
		Notes:      toVMScanNotes(vr.Notes),
		//TODO: Dynamically read the operating system from... something
		OperatingSystem: "linux",
		//TODO: Should this be read from the report?
		ScannerVersion: "Scanner V4",
	}
}

func toImageScanComponents(vr *v4.VulnerabilityReport) []*storage.EmbeddedImageScanComponent {
	packages := vr.GetContents().GetPackages()
	result := make([]*storage.EmbeddedImageScanComponent, 0, len(packages))

	for _, pkg := range packages {
		result = append(result, &storage.EmbeddedImageScanComponent{
			Name:    pkg.GetName(),
			Version: pkg.GetVersion(),
			Vulns:   getVMPackageVulns(pkg.GetId(), vr),
		})
	}
	return result
}

func getVMPackageVulns(packageID string, vr *v4.VulnerabilityReport) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0)
	mapping, ok := vr.GetPackageVulnerabilities()[packageID]
	if !ok {
		return vulns
	}

	processedVulns := set.NewStringSet()
	for _, vulnID := range mapping.GetValues() {
		if !processedVulns.Add(vulnID) {
			continue
		}
		vulnerability, ok := vr.Vulnerabilities[vulnID]
		if !ok {
			log.Debugf("VM package %s contains unknown vulnerability ID %s", packageID, vulnID)
			continue
		}
		vulns = append(vulns, convertVMVulnerability(vulnerability))
	}
	return vulns
}

func convertVMVulnerability(v *v4.VulnerabilityReport_Vulnerability) *storage.EmbeddedVulnerability {
	converted := &storage.EmbeddedVulnerability{
		Cve:               v.GetName(),
		Summary:           v.GetDescription(),
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		Severity:          normalizedSeverity(v.GetNormalizedSeverity()),
		//TODO: link is deprecated - do we need to have it at all?
		Link:        v.GetLink(),
		PublishedOn: v.GetIssued(),
		Advisory:    convertAdvisory(v.GetAdvisory()),
	}

	if err := setScoresAndScoreVersions(converted, v.GetCvssMetrics()); err != nil {
		log.Warnf("Failed to set CVSS scores for vulnerability %s: %v", v.GetName(), err)
	}
	maybeOverwriteSeverity(converted)

	if v.GetFixedInVersion() != "" {
		converted.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: v.GetFixedInVersion(),
		}
	}

	return converted
}

func convertAdvisory(advisory *v4.VulnerabilityReport_Advisory) *storage.Advisory {
	if advisory == nil {
		return nil
	}
	return &storage.Advisory{
		Name: advisory.GetName(),
		Link: advisory.GetLink(),
	}
}

func toVMScanNotes(notes []v4.VulnerabilityReport_Note) []storage.VirtualMachineScan_Note {
	convertedNotes := make([]storage.VirtualMachineScan_Note, 0, len(notes))
	for _, n := range notes {
		switch n {
		//TODO: Add more vm notes to the enum
		case v4.VulnerabilityReport_NOTE_OS_UNKNOWN:
			convertedNotes = append(convertedNotes, storage.VirtualMachineScan_OS_UNAVAILABLE)
		case v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED:
			convertedNotes = append(convertedNotes, storage.VirtualMachineScan_OS_UNAVAILABLE)
		case v4.VulnerabilityReport_NOTE_UNSPECIFIED:
			convertedNotes = append(convertedNotes, storage.VirtualMachineScan_UNSET)
		default:
			log.Warnf("Unknown VM vulnerability report note: %s", n.String())
		}
	}
	return convertedNotes
}
