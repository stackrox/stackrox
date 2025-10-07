package scannerv4

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

// ToVirtualMachineScan converts a scan report to the format needed to enrich a virtual machine with scan data.
func ToVirtualMachineScan(r *v4.VulnerabilityReport) *storage.VirtualMachineScan {
	return &storage.VirtualMachineScan{
		ScanTime: protocompat.TimestampNow(),
		// TODO: find an actual operating system source
		OperatingSystem: "",
		Notes:           toVirtualMachineScanNotes(r.GetNotes()),
		Components:      toVirtualMachineComponents(r),
	}
}

func toVirtualMachineScanNotes(notes []v4.VulnerabilityReport_Note) []storage.VirtualMachineScan_Note {
	result := make([]storage.VirtualMachineScan_Note, 0, len(notes))
	for _, note := range notes {
		switch note {
		case v4.VulnerabilityReport_NOTE_UNSPECIFIED:
			result = append(result, storage.VirtualMachineScan_UNSET)
		case v4.VulnerabilityReport_NOTE_OS_UNKNOWN:
			result = append(result, storage.VirtualMachineScan_OS_UNKNOWN)
		case v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED:
			result = append(result, storage.VirtualMachineScan_OS_UNSUPPORTED)
		default:
			result = append(result, storage.VirtualMachineScan_UNSET)
		}
	}
	return result
}

func toVirtualMachineComponents(r *v4.VulnerabilityReport) []*storage.EmbeddedVirtualMachineScanComponent {
	packages := r.GetContents().GetPackages()
	result := make([]*storage.EmbeddedVirtualMachineScanComponent, 0, len(packages))
	for _, pkg := range packages {
		vulnerabilityIDs := r.GetPackageVulnerabilities()[pkg.GetId()].GetValues()
		vulnerabilitiesByID := r.GetVulnerabilities()
		component := &storage.EmbeddedVirtualMachineScanComponent{
			Name:    pkg.GetName(),
			Version: pkg.GetVersion(),
			// Architecture ?
			Vulnerabilities: toVirtualMachineScanComponentVulnerabilities(vulnerabilitiesByID, vulnerabilityIDs),
		}
		result = append(result, component)
	}
	return result
}

func toVirtualMachineScanComponentVulnerabilities(
	vulnerabilitiesByID map[string]*v4.VulnerabilityReport_Vulnerability,
	vulnerabilityIDs []string,
) []*storage.VirtualMachineVulnerability {
	embeddedVulns := vulnerabilities(vulnerabilitiesByID, vulnerabilityIDs)
	result := make([]*storage.VirtualMachineVulnerability, 0, len(embeddedVulns))
	for _, vuln := range embeddedVulns {
		resultVuln := &storage.VirtualMachineVulnerability{
			CveBaseInfo: &storage.VirtualMachineCVEInfo{
				Cve:          vuln.GetCve(),
				Summary:      vuln.GetSummary(),
				Link:         vuln.GetLink(),
				PublishedOn:  vuln.GetPublishedOn(),
				LastModified: vuln.GetLastModified(),
				CvssMetrics:  vuln.GetCvssMetrics(),
				Epss:         toVirtualMachineEPSS(vuln.GetEpss()),
				Advisory:     toVirtualMachineAdvisory(vuln.GetAdvisory()),
			},
			Severity: vuln.GetSeverity(),
			Cvss:     vuln.GetCvss(),
		}
		if vuln.GetSetFixedBy() != nil {
			resultVuln.SetFixedBy = &storage.VirtualMachineVulnerability_FixedBy{
				FixedBy: vuln.GetFixedBy(),
			}
		}
		result = append(result, resultVuln)
	}
	return result
}

func toVirtualMachineAdvisory(
	advisory *storage.Advisory,
) *storage.VirtualMachineAdvisory {
	if advisory == nil {
		return nil
	}
	return &storage.VirtualMachineAdvisory{
		Name: advisory.GetName(),
		Link: advisory.GetLink(),
	}
}

func toVirtualMachineEPSS(epss *storage.EPSS) *storage.VirtualMachineEPSS {
	if epss == nil {
		return nil
	}
	return &storage.VirtualMachineEPSS{
		EpssProbability: epss.GetEpssProbability(),
		EpssPercentile:  epss.GetEpssPercentile(),
	}
}
