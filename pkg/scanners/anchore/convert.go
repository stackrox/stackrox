package anchore

import (
	"strings"
	"time"

	anchoreClient "github.com/stackrox/anchore-client/client"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
)

func getSeverity(s string) float32 {
	switch strings.ToLower(s) {
	case "critical":
		return 9.5
	case "high":
		return 8.5
	case "medium":
		return 5.5
	case "low":
		return 2.0
	default:
		return 0.0
	}
}

func convertImageScan(i *anchoreClient.AnchoreImage, packages []anchoreClient.ContentPackageResponseContent, vulns []anchoreClient.Vulnerability) *storage.ImageScan {
	t, err := time.Parse(time.RFC3339, i.AnalyzedAt)
	if err != nil {
		log.Error(err)
	}
	protoTS := protoconv.ConvertTimeToTimestamp(t)
	return &storage.ImageScan{
		ScanTime:   protoTS,
		Components: stitchPackagesAndVulns(packages, vulns),
	}
}

func convertPackageToComponent(pkg anchoreClient.ContentPackageResponseContent) *storage.EmbeddedImageScanComponent {
	return &storage.EmbeddedImageScanComponent{
		Name:    pkg.Package_,
		Version: pkg.Version,
		License: &storage.License{
			Name: pkg.License,
		},
	}
}

func convertVulnToProtoVuln(vuln anchoreClient.Vulnerability) *storage.EmbeddedVulnerability {
	if strings.EqualFold(vuln.Fix, "none") {
		vuln.Fix = ""
	}

	embeddedVuln := &storage.EmbeddedVulnerability{
		Cve:     vuln.Vuln,
		Summary: "Follow the link for CVE summary",
		Link:    vuln.Url,
		SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
			FixedBy: vuln.Fix,
		},
		Cvss: getSeverity(vuln.Severity),
	}

	if len(vuln.NVDData) != 0 {
		if cvssV3 := vuln.NVDData[0].CVSSV3; cvssV3 != nil && cvssV3.Base != -1 {
			embeddedVuln.Vectors = &storage.EmbeddedVulnerability_CvssV3{
				CvssV3: &storage.CVSSV3{
					ImpactScore:         float32(cvssV3.Impact),
					ExploitabilityScore: float32(cvssV3.Exploitability),
				},
			}
			embeddedVuln.Cvss = float32(cvssV3.Base)
			embeddedVuln.ScoreVersion = storage.EmbeddedVulnerability_V3
		} else if cvssV2 := vuln.NVDData[0].CVSSV2; cvssV2 != nil && cvssV2.Base != -1 {
			embeddedVuln.Cvss = float32(vuln.NVDData[0].CVSSV2.Base)
		}
	}
	return embeddedVuln
}

type componentKey struct {
	pkg, version string
}

func stitchPackagesAndVulns(packages []anchoreClient.ContentPackageResponseContent, vulns []anchoreClient.Vulnerability) []*storage.EmbeddedImageScanComponent {
	componentMap := make(map[componentKey]*storage.EmbeddedImageScanComponent)
	for _, p := range packages {
		componentMap[componentKey{pkg: p.Package_, version: p.Version}] = convertPackageToComponent(p)
	}

	for _, v := range vulns {
		key := componentKey{pkg: v.PackageName, version: v.PackageVersion}
		_, ok := componentMap[key]
		if !ok {
			componentMap[key] = &storage.EmbeddedImageScanComponent{
				Name:    v.PackageName,
				Version: v.PackageVersion,
			}
		}
		component := componentMap[key]
		component.Vulns = append(component.Vulns, convertVulnToProtoVuln(v))
	}
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(componentMap))
	for _, v := range componentMap {
		components = append(components, v)
	}
	return components
}
