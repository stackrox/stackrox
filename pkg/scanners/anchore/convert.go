package anchore

import (
	"strings"
	"time"

	anchoreClient "github.com/stackrox/anchore-client/client"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/set"
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

func convertPackageToComponentValue(pkg anchoreClient.ContentPackageResponseContent) *componentValue {
	return newComponentValue(&storage.EmbeddedImageScanComponent{
		Name:    pkg.Package_,
		Version: pkg.Version,
		License: &storage.License{
			Name: pkg.License,
		},
	})
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
		Cvss:              getSeverity(vuln.Severity),
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
	}

	if len(vuln.NVDData) != 0 {
		if cvssV2 := vuln.NVDData[0].CVSSV2; cvssV2 != nil && cvssV2.Base != -1 {
			embeddedVuln.CvssV2 = &storage.CVSSV2{
				Score:               float32(cvssV2.Base),
				ImpactScore:         float32(cvssV2.Impact),
				ExploitabilityScore: float32(cvssV2.Exploitability),
			}
			embeddedVuln.Cvss = float32(vuln.NVDData[0].CVSSV2.Base)
			embeddedVuln.ScoreVersion = storage.EmbeddedVulnerability_V2
		}
		if cvssV3 := vuln.NVDData[0].CVSSV3; cvssV3 != nil && cvssV3.Base != -1 {
			embeddedVuln.CvssV3 = &storage.CVSSV3{
				ImpactScore:         float32(cvssV3.Impact),
				ExploitabilityScore: float32(cvssV3.Exploitability),
				Score:               float32(cvssV3.Base),
			}
			embeddedVuln.Cvss = float32(cvssV3.Base)
			embeddedVuln.ScoreVersion = storage.EmbeddedVulnerability_V3
		}
	}
	return embeddedVuln
}

type componentKey struct {
	pkg, version string
}

func newComponentValue(c *storage.EmbeddedImageScanComponent) *componentValue {
	s := set.NewStringSet()
	return &componentValue{
		component: c,
		vulnSet:   &s,
	}
}

type componentValue struct {
	component *storage.EmbeddedImageScanComponent
	vulnSet   *set.StringSet
}

func (c *componentValue) addVuln(vulnerability anchoreClient.Vulnerability) {
	if added := c.vulnSet.Add(vulnerability.Vuln); !added {
		return
	}
	c.component.Vulns = append(c.component.Vulns, convertVulnToProtoVuln(vulnerability))
}

func stitchPackagesAndVulns(packages []anchoreClient.ContentPackageResponseContent, vulns []anchoreClient.Vulnerability) []*storage.EmbeddedImageScanComponent {
	componentMap := make(map[componentKey]*componentValue)
	for _, p := range packages {
		componentMap[componentKey{pkg: p.Package_, version: p.Version}] = convertPackageToComponentValue(p)
	}

	for _, v := range vulns {
		key := componentKey{pkg: v.PackageName, version: v.PackageVersion}
		_, ok := componentMap[key]
		if !ok {
			componentMap[key] = newComponentValue(&storage.EmbeddedImageScanComponent{
				Name:    v.PackageName,
				Version: v.PackageVersion,
			})
		}
		component := componentMap[key]
		component.addVuln(v)
	}
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(componentMap))
	for _, v := range componentMap {
		components = append(components, v.component)
	}
	return components
}
