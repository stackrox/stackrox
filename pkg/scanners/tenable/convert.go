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
