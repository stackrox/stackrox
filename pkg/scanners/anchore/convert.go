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

func convertPackageToComponent(pkg anchoreClient.ContentPackageResponseContent) *storage.ImageScanComponent {
	return &storage.ImageScanComponent{
		Name:    pkg.Package_,
		Version: pkg.Version,
		License: &storage.License{
			Name: pkg.License,
		},
	}
}

func convertVulnToProtoVuln(vuln anchoreClient.Vulnerability) *storage.Vulnerability {
	if strings.EqualFold(vuln.Fix, "none") {
		vuln.Fix = ""
	}
	return &storage.Vulnerability{
		Cve:     vuln.Vuln,
		Cvss:    getSeverity(vuln.Severity),
		Summary: "Follow the link for CVE summary",
		Link:    vuln.Url,
		SetFixedBy: &storage.Vulnerability_FixedBy{
			FixedBy: vuln.Fix,
		},
	}
}

type componentKey struct {
	pkg, version string
}

func stitchPackagesAndVulns(packages []anchoreClient.ContentPackageResponseContent, vulns []anchoreClient.Vulnerability) []*storage.ImageScanComponent {
	componentMap := make(map[componentKey]*storage.ImageScanComponent)
	for _, p := range packages {
		componentMap[componentKey{pkg: p.Package_, version: p.Version}] = convertPackageToComponent(p)
	}

	for _, v := range vulns {
		key := componentKey{pkg: v.PackageName, version: v.PackageVersion}
		_, ok := componentMap[key]
		if !ok {
			componentMap[key] = &storage.ImageScanComponent{
				Name:    v.PackageName,
				Version: v.PackageVersion,
			}
		}
		component := componentMap[key]
		component.Vulns = append(component.Vulns, convertVulnToProtoVuln(v))
	}
	components := make([]*storage.ImageScanComponent, 0, len(componentMap))
	for _, v := range componentMap {
		components = append(components, v)
	}
	return components
}
