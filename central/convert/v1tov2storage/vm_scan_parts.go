package v1tov2storage

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// ScanPartsFromV1Scan converts a v1 VirtualMachineScan into normalized v2
// VMScanParts (scan, components, CVEs) ready for upsert into v2 datastores.
func ScanPartsFromV1Scan(vmID string, scan *storage.VirtualMachineScan) *common.VMScanParts {
	if scan == nil {
		return nil
	}

	scanID := uuid.NewV4().String()

	var (
		components       []*storage.VirtualMachineComponentV2
		cves             []*storage.VirtualMachineCVEV2
		topCvss          float32
		sourceComponents = scan.GetComponents()
	)

	for _, comp := range sourceComponents {
		componentID := uuid.NewV4().String()
		var componentCVEs []*storage.VirtualMachineCVEV2

		for _, vuln := range comp.GetVulnerabilities() {
			cve := convertVulnerability(vmID, componentID, vuln)
			componentCVEs = append(componentCVEs, cve)
			if cve.GetPreferredCvss() > topCvss {
				topCvss = cve.GetPreferredCvss()
			}
		}

		cves = append(cves, componentCVEs...)

		var compTopCvss float32
		for _, c := range componentCVEs {
			if c.GetPreferredCvss() > compTopCvss {
				compTopCvss = c.GetPreferredCvss()
			}
		}

		components = append(components, &storage.VirtualMachineComponentV2{
			Id:              componentID,
			VmScanId:        scanID,
			Name:            comp.GetName(),
			Version:         comp.GetVersion(),
			Source:          comp.GetSource(),
			OperatingSystem: scan.GetOperatingSystem(),
			SetTopCvss: &storage.VirtualMachineComponentV2_TopCvss{
				TopCvss: compTopCvss,
			},
			FixedBy:  highestFixedBy(comp.GetVulnerabilities()),
			CveCount: int32(len(componentCVEs)),
		})
	}

	scanNotes := convertScanNotes(scan.GetNotes())

	return &common.VMScanParts{
		Scan: &storage.VirtualMachineScanV2{
			Id:       scanID,
			VmV2Id:   vmID,
			ScanOs:   scan.GetOperatingSystem(),
			ScanTime: scan.GetScanTime(),
			TopCvss:  topCvss,
			Notes:    scanNotes,
		},
		Components:       components,
		CVEs:             cves,
		SourceComponents: sourceComponents,
	}
}

func convertVulnerability(vmID, componentID string, vuln *storage.VirtualMachineVulnerability) *storage.VirtualMachineCVEV2 {
	info := vuln.GetCveBaseInfo()

	preferredCvss, preferredVersion := preferredCvssVersion(info)
	nvdCvss, nvdVersion := nvdCvssScore(info)
	impactScore := extractImpactScore(info)

	cve := &storage.VirtualMachineCVEV2{
		Id:                   uuid.NewV4().String(),
		VmV2Id:               vmID,
		VmComponentId:        componentID,
		CveBaseInfo:          convertCVEBaseInfo(info),
		PreferredCvss:        preferredCvss,
		PreferredCvssVersion: preferredVersion,
		Severity:             vuln.GetSeverity(),
		ImpactScore:          impactScore,
		Nvdcvss:              nvdCvss,
		NvdScoreVersion:      nvdVersion,
	}

	if fixedBy := vuln.GetFixedBy(); fixedBy != "" {
		cve.IsFixable = true
		cve.HasFixedBy = &storage.VirtualMachineCVEV2_FixedBy{FixedBy: fixedBy}
	}

	if epss := info.GetEpss(); epss != nil {
		cve.EpssProbability = epss.GetEpssProbability()
	}

	if adv := info.GetAdvisory(); adv != nil {
		cve.Advisory = &storage.Advisory{
			Name: adv.GetName(),
			Link: adv.GetLink(),
		}
	}

	return cve
}

func convertCVEBaseInfo(info *storage.VirtualMachineCVEInfo) *storage.CVEInfo {
	if info == nil {
		return nil
	}

	var refs []*storage.CVEInfo_Reference
	for _, r := range info.GetReferences() {
		refs = append(refs, &storage.CVEInfo_Reference{
			URI:  r.GetURI(),
			Tags: r.GetTags(),
		})
	}

	return &storage.CVEInfo{
		Cve:          info.GetCve(),
		Summary:      info.GetSummary(),
		Link:         info.GetLink(),
		PublishedOn:  info.GetPublishedOn(),
		CreatedAt:    info.GetCreatedAt(),
		LastModified: info.GetLastModified(),
		CvssMetrics:  info.GetCvssMetrics(),
		References:   refs,
	}
}

func convertScanNotes(notes []storage.VirtualMachineScan_Note) []storage.VirtualMachineScanV2_Note {
	if len(notes) == 0 {
		return nil
	}
	out := make([]storage.VirtualMachineScanV2_Note, 0, len(notes))
	for _, n := range notes {
		switch n {
		case storage.VirtualMachineScan_OS_UNKNOWN:
			out = append(out, storage.VirtualMachineScanV2_OS_UNKNOWN)
		case storage.VirtualMachineScan_OS_UNSUPPORTED:
			out = append(out, storage.VirtualMachineScanV2_OS_UNSUPPORTED)
		default:
			out = append(out, storage.VirtualMachineScanV2_UNSET)
		}
	}
	return out
}

// preferredCvssVersion picks the preferred CVSS score and version (V3 > V2).
func preferredCvssVersion(info *storage.VirtualMachineCVEInfo) (float32, storage.CvssScoreVersion) {
	if info == nil {
		return 0, storage.CvssScoreVersion_UNKNOWN_VERSION
	}

	for _, m := range info.GetCvssMetrics() {
		if m.GetSource() == storage.Source_SOURCE_RED_HAT {
			if v3 := m.GetCvssv3(); v3 != nil {
				return v3.GetScore(), storage.CvssScoreVersion_V3
			}
			if v2 := m.GetCvssv2(); v2 != nil {
				return v2.GetScore(), storage.CvssScoreVersion_V2
			}
		}
	}

	// Fall back to any metric with a score.
	for _, m := range info.GetCvssMetrics() {
		if v3 := m.GetCvssv3(); v3 != nil {
			return v3.GetScore(), storage.CvssScoreVersion_V3
		}
		if v2 := m.GetCvssv2(); v2 != nil {
			return v2.GetScore(), storage.CvssScoreVersion_V2
		}
	}

	return 0, storage.CvssScoreVersion_UNKNOWN_VERSION
}

// nvdCvssScore extracts the NVD-specific CVSS score from CVSS metrics.
func nvdCvssScore(info *storage.VirtualMachineCVEInfo) (float32, storage.CvssScoreVersion) {
	if info == nil {
		return 0, storage.CvssScoreVersion_UNKNOWN_VERSION
	}

	for _, m := range info.GetCvssMetrics() {
		if m.GetSource() == storage.Source_SOURCE_NVD {
			if v3 := m.GetCvssv3(); v3 != nil {
				return v3.GetScore(), storage.CvssScoreVersion_V3
			}
			if v2 := m.GetCvssv2(); v2 != nil {
				return v2.GetScore(), storage.CvssScoreVersion_V2
			}
		}
	}

	return 0, storage.CvssScoreVersion_UNKNOWN_VERSION
}

// extractImpactScore extracts the impact score from the preferred CVSS V3 metric.
func extractImpactScore(info *storage.VirtualMachineCVEInfo) float32 {
	if info == nil {
		return 0
	}

	for _, m := range info.GetCvssMetrics() {
		if m.GetSource() == storage.Source_SOURCE_RED_HAT {
			if v3 := m.GetCvssv3(); v3 != nil {
				return v3.GetImpactScore()
			}
		}
	}

	for _, m := range info.GetCvssMetrics() {
		if v3 := m.GetCvssv3(); v3 != nil {
			return v3.GetImpactScore()
		}
	}

	return 0
}

// highestFixedBy finds the highest fixed-by version across all vulnerabilities.
func highestFixedBy(vulns []*storage.VirtualMachineVulnerability) string {
	var highest string
	for _, v := range vulns {
		if fb := v.GetFixedBy(); fb != "" {
			if highest == "" || compareVersionSegments(fb, highest) > 0 {
				highest = fb
			}
		}
	}
	return highest
}

// compareVersionSegments compares two dot-separated version strings numerically
// segment by segment. Returns negative if a < b, 0 if equal, positive if a > b.
func compareVersionSegments(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aVal, bVal int
		if i < len(aParts) {
			aVal, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bVal, _ = strconv.Atoi(bParts[i])
		}
		if aVal != bVal {
			return aVal - bVal
		}
	}
	return 0
}
