package v1tov2storage

import (
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// ScanPartsFromV1Scan converts a v1 VirtualMachineScan into normalized v2 scan
// parts (scan, components, CVEs) suitable for UpsertScan.
func ScanPartsFromV1Scan(vmID string, scan *storage.VirtualMachineScan) common.VMScanParts {
	if scan == nil {
		return common.VMScanParts{}
	}

	scanID := uuid.NewV4().String()
	scanV2 := &storage.VirtualMachineScanV2{
		Id:       scanID,
		VmV2Id:   vmID,
		ScanOs:   scan.GetOperatingSystem(),
		ScanTime: scan.GetScanTime(),
	}

	var components []*storage.VirtualMachineComponentV2
	var cves []*storage.VirtualMachineCVEV2

	for _, comp := range scan.GetComponents() {
		componentID := uuid.NewV4().String()

		fixedBy := highestFixedBy(comp.GetVulnerabilities())

		componentV2 := &storage.VirtualMachineComponentV2{
			Id:       componentID,
			VmScanId: scanID,
			Name:     comp.GetName(),
			Version:  comp.GetVersion(),
			Source:   comp.GetSource(),
			FixedBy:  fixedBy,
			CveCount: int32(len(comp.GetVulnerabilities())),
		}
		if cvss := comp.GetTopCvss(); cvss != 0 {
			componentV2.SetTopCvss = &storage.VirtualMachineComponentV2_TopCvss{TopCvss: cvss}
		}
		components = append(components, componentV2)

		for _, vuln := range comp.GetVulnerabilities() {
			cveV2 := convertVulnerability(vmID, componentID, vuln)
			cves = append(cves, cveV2)
		}
	}

	return common.VMScanParts{
		Scan:             scanV2,
		Components:       components,
		CVEs:             cves,
		SourceComponents: scan.GetComponents(),
	}
}

func convertVulnerability(vmID, componentID string, vuln *storage.VirtualMachineVulnerability) *storage.VirtualMachineCVEV2 {
	cveInfo := vuln.GetCveBaseInfo()

	preferredCvss := vuln.GetCvss()
	preferredVersion := preferredCvssVersion(cveInfo)

	var scoreVersion storage.CVEInfo_ScoreVersion
	var impactScore float32
	var nvdCvss float32
	nvdVersion := storage.CvssScoreVersion_UNKNOWN_VERSION

	// Extract CvssV2/V3 and NVD scores from cvss_metrics.
	var cvssV2 *storage.CVSSV2
	var cvssV3 *storage.CVSSV3
	for _, score := range cveInfo.GetCvssMetrics() {
		if score.GetCvssv3() != nil && cvssV3 == nil {
			cvssV3 = score.GetCvssv3()
		}
		if score.GetCvssv2() != nil && cvssV2 == nil {
			cvssV2 = score.GetCvssv2()
		}
		if score.GetSource() == storage.Source_SOURCE_NVD {
			if score.GetCvssv3() != nil {
				nvdCvss = score.GetCvssv3().GetScore()
				nvdVersion = storage.CvssScoreVersion_V3
			} else if score.GetCvssv2() != nil {
				nvdCvss = score.GetCvssv2().GetScore()
				nvdVersion = storage.CvssScoreVersion_V2
			}
		}
	}

	if cvssV3 != nil {
		scoreVersion = storage.CVEInfo_V3
		impactScore = cvssV3.GetImpactScore()
	} else if cvssV2 != nil {
		scoreVersion = storage.CVEInfo_V2
		impactScore = cvssV2.GetImpactScore()
	}

	fixedBy := vuln.GetFixedBy()

	cveV2 := &storage.VirtualMachineCVEV2{
		Id:            uuid.NewV4().String(),
		VmV2Id:        vmID,
		VmComponentId: componentID,
		CveBaseInfo: &storage.CVEInfo{
			Cve:          cveInfo.GetCve(),
			Summary:      cveInfo.GetSummary(),
			Link:         cveInfo.GetLink(),
			PublishedOn:  cveInfo.GetPublishedOn(),
			CreatedAt:    cveInfo.GetCreatedAt(),
			LastModified: cveInfo.GetLastModified(),
			CvssV2:       cvssV2,
			CvssV3:       cvssV3,
			CvssMetrics:  cveInfo.GetCvssMetrics(),
			ScoreVersion: scoreVersion,
		},
		PreferredCvss:        preferredCvss,
		PreferredCvssVersion: preferredVersion,
		Severity:             vuln.GetSeverity(),
		ImpactScore:          impactScore,
		Nvdcvss:              nvdCvss,
		NvdScoreVersion:      nvdVersion,
		IsFixable:            fixedBy != "",
		EpssProbability:      cveInfo.GetEpss().GetEpssProbability(),
	}

	if fixedBy != "" {
		cveV2.HasFixedBy = &storage.VirtualMachineCVEV2_FixedBy{
			FixedBy: fixedBy,
		}
	}

	if adv := cveInfo.GetAdvisory(); adv != nil {
		cveV2.Advisory = &storage.Advisory{
			Name: adv.GetName(),
			Link: adv.GetLink(),
		}
	}

	return cveV2
}

// highestFixedBy returns the highest fixed_by version string across all vulns,
// or empty if none are fixable.
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

// compareVersionSegments compares two version strings by splitting on "." and
// comparing each segment numerically. Non-numeric segments fall back to
// lexicographic comparison. Returns <0, 0, or >0.
func compareVersionSegments(a, b string) int {
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")
	n := len(partsA)
	if len(partsB) > n {
		n = len(partsB)
	}
	for i := 0; i < n; i++ {
		var sa, sb string
		if i < len(partsA) {
			sa = partsA[i]
		}
		if i < len(partsB) {
			sb = partsB[i]
		}
		na, errA := strconv.Atoi(sa)
		nb, errB := strconv.Atoi(sb)
		if errA == nil && errB == nil {
			if na != nb {
				return na - nb
			}
			continue
		}
		if sa != sb {
			if sa < sb {
				return -1
			}
			return 1
		}
	}
	return 0
}

// preferredCvssVersion determines the preferred CVSS score version from the
// vulnerability's cvss_metrics. V3 is preferred over V2.
func preferredCvssVersion(cveInfo *storage.VirtualMachineCVEInfo) storage.CvssScoreVersion {
	hasV2 := false
	for _, score := range cveInfo.GetCvssMetrics() {
		if score.GetCvssv3() != nil {
			return storage.CvssScoreVersion_V3
		}
		if score.GetCvssv2() != nil {
			hasV2 = true
		}
	}
	if hasV2 {
		return storage.CvssScoreVersion_V2
	}
	return storage.CvssScoreVersion_UNKNOWN_VERSION
}
