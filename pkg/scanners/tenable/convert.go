package tenable

import (
	"strconv"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/scans"
)

func convertNVDFindingsAndPackagesToComponents(findings []*finding, packages []pkg) (components []*v1.ImageScanComponent) {
	packagesToVulnerabilities := make(map[pkg][]*v1.Vulnerability)
	for _, finding := range findings {
		// If there's an error then we are going to take the default value of 0.0 anyways
		cvssScore, err := strconv.ParseFloat(finding.NVDFinding.CVSSScore, 32)
		if err != nil {
			log.Warn(err)
		}
		vulnerability := &v1.Vulnerability{
			Cvss:    float32(cvssScore),
			Cve:     finding.NVDFinding.CVE,
			Summary: finding.NVDFinding.Description,
			Link:    scans.GetVulnLink(finding.NVDFinding.CVE),
			CvssV2:  convertCVSS(finding.NVDFinding),
		}
		for _, affectedPackage := range finding.Packages {
			packagesToVulnerabilities[affectedPackage] = append(packagesToVulnerabilities[affectedPackage], vulnerability)
		}
	}
	for _, p := range packages {
		if _, ok := packagesToVulnerabilities[p]; !ok {
			packagesToVulnerabilities[p] = nil
		}
	}
	for p, vulns := range packagesToVulnerabilities {
		components = append(components, &v1.ImageScanComponent{
			Name:    p.Name,
			Version: p.Version,
			Vulns:   vulns,
		})
	}
	return components
}

func convertScanToImageScan(image *v1.Image, s *scanResult) *v1.ImageScan {
	completedAt, err := ptypes.TimestampProto(s.UpdatedAt)
	if err != nil {
		log.Error(err)
	}
	components := convertNVDFindingsAndPackagesToComponents(s.Findings, s.InstalledPackages)
	return &v1.ImageScan{
		ScanTime:   completedAt,
		Components: components,
	}
}

func getImpact(i string) v1.CVSSV2_Impact {
	switch i {
	case "None":
		return v1.CVSSV2_IMPACT_NONE
	case "Partial":
		return v1.CVSSV2_IMPACT_PARTIAL
	case "Complete":
		return v1.CVSSV2_IMPACT_COMPLETE
	default:
		log.Errorf("Impact could not parse: %v", i)
		return v1.CVSSV2_IMPACT_COMPLETE
	}
}

func convertCVSS(nvd nvdFinding) *v1.CVSSV2 {
	var cvss v1.CVSSV2
	switch nvd.AccessVector {
	case "Network":
		cvss.AttackVector = v1.CVSSV2_ATTACK_NETWORK
	case "Adjacent":
		cvss.AttackVector = v1.CVSSV2_ATTACK_ADJACENT
	case "Local":
		cvss.AttackVector = v1.CVSSV2_ATTACK_LOCAL
	default:
		log.Errorf("Could not parse access vector %v", nvd.AccessVector)
		cvss.AttackVector = v1.CVSSV2_ATTACK_NETWORK
	}
	switch nvd.Auth {
	case "None required":
		cvss.Authentication = v1.CVSSV2_AUTH_NONE
	case "Single":
		cvss.Authentication = v1.CVSSV2_AUTH_SINGLE
	case "Multiple":
		cvss.Authentication = v1.CVSSV2_AUTH_MULTIPLE
	default:
		log.Errorf("Could not parse auth vector %v", nvd.Auth)
		cvss.Authentication = v1.CVSSV2_AUTH_MULTIPLE
	}
	switch nvd.AccessComplexity {
	case "Low":
		cvss.AccessComplexity = v1.CVSSV2_ACCESS_LOW
	case "Medium":
		cvss.AccessComplexity = v1.CVSSV2_ACCESS_MEDIUM
	case "High":
		cvss.AccessComplexity = v1.CVSSV2_ACCESS_HIGH
	default:
		log.Errorf("Could not parse access complexity %v", nvd.AccessComplexity)
		cvss.AccessComplexity = v1.CVSSV2_ACCESS_HIGH
	}
	cvss.Availability = getImpact(nvd.AvailabilityImpact)
	cvss.Confidentiality = getImpact(nvd.ConfidentialityImpact)
	cvss.Integrity = getImpact(nvd.IntegrityImpact)
	return &cvss
}
