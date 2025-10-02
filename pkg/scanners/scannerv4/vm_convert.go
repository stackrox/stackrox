package scannerv4

import (
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	pkgCVSS "github.com/stackrox/rox/pkg/cvss"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
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
	result := make([]*storage.VirtualMachineVulnerability, 0, len(vulnerabilityIDs))
	processedVulnerabilityIDs := set.NewStringSet()
	for _, vulnerabilityID := range vulnerabilityIDs {
		if shouldProcess := processedVulnerabilityIDs.Add(vulnerabilityID); !shouldProcess {
			continue
		}
		reportVulnerability, found := vulnerabilitiesByID[vulnerabilityID]
		if !found {
			continue
		}
		vulnerability := &storage.VirtualMachineVulnerability{
			CveBaseInfo: &storage.VirtualMachineCVEInfo{
				Cve:         reportVulnerability.GetName(),
				Summary:     reportVulnerability.GetDescription(),
				Link:        link(reportVulnerability.GetLink()),
				PublishedOn: reportVulnerability.GetIssued(),
				Advisory:    toVirtualMachineAdvisory(reportVulnerability.GetAdvisory()),
			},
			Severity: normalizedSeverity(reportVulnerability.GetNormalizedSeverity()),
		}
		vulnerabilityMetrics := reportVulnerability.GetCvssMetrics()
		if err := setVirtualMachineScoresAndScoreVersions(vulnerability, vulnerabilityMetrics); err != nil {
			utils.Should(err)
		}

		if reportVulnerability.GetFixedInVersion() != "" {
			vulnerability.SetFixedBy = &storage.VirtualMachineVulnerability_FixedBy{
				FixedBy: reportVulnerability.GetFixedInVersion(),
			}
		}

		result = append(result, vulnerability)
	}
	return result
}

func toVirtualMachineAdvisory(
	advisory *v4.VulnerabilityReport_Advisory,
) *storage.VirtualMachineAdvisory {
	if advisory == nil {
		return nil
	}
	return &storage.VirtualMachineAdvisory{
		Name: advisory.GetName(),
		Link: advisory.GetLink(),
	}
}

func setVirtualMachineScoresAndScoreVersions(
	vulnerability *storage.VirtualMachineVulnerability,
	cvssMetrics []*v4.VulnerabilityReport_Vulnerability_CVSS,
) error {
	if vulnerability == nil || vulnerability.CveBaseInfo == nil {
		return errox.InvalidArgs.CausedBy("Cannot enrich CVSS information on a nil vulnerability object")
	}
	severity := vulnerability.GetSeverity()
	cvssV3Propagated := false
	if len(cvssMetrics) == 0 {
		return nil
	}
	cve := vulnerability.GetCveBaseInfo().GetCve()
	errList := errorhelpers.NewErrorList("failed to get CVSS metrics")
	var scores []*storage.CVSSScore
	for _, cvss := range cvssMetrics {
		score := &storage.CVSSScore{
			Source: CVSSSource(cvss.GetSource()),
			Url:    cvss.GetUrl(),
		}
		if cvss.GetV2() != nil {
			_, cvssV2, v2Err := toCVSSV2Scores(cvss, cve)
			if v2Err == nil && cvssV2 != nil {
				score.CvssScore = &storage.CVSSScore_Cvssv2{Cvssv2: cvssV2}
				if severity == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY && !cvssV3Propagated {
					vulnerability.Severity = pkgCVSS.ConvertCVSSV2SeverityToVulnerabilitySeverity(cvssV2.GetSeverity())
				}
			} else {
				errList.AddError(v2Err)
			}
		}
		if cvss.GetV3() != nil {
			_, cvssV3, v3Err := toCVSSV3Scores(cvss, cve)
			if v3Err == nil && cvssV3 != nil {
				score.CvssScore = &storage.CVSSScore_Cvssv3{Cvssv3: cvssV3}
				// CVSS metrics has maximum two entries, one from NVD, one from Rox updater if available
				if len(cvssMetrics) == 1 || (len(cvssMetrics) > 1 && cvss.Source != v4.VulnerabilityReport_Vulnerability_CVSS_SOURCE_NVD) {
					vulnerability.CveBaseInfo.Link = cvss.GetUrl()
					if severity == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
						vulnerability.Severity = pkgCVSS.ConvertCVSSV3SeverityToVulnerabilitySeverity(cvssV3.GetSeverity())
						cvssV3Propagated = true
					}
				}
			} else {
				errList.AddError(v3Err)
			}
		}
		if score.CvssScore != nil {
			scores = append(scores, score)
		}
	}
	if len(scores) > 0 {
		vulnerability.CveBaseInfo.CvssMetrics = scores
	}
	return errList.ToError()
}
