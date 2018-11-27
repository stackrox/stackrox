package dtr

import (
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/scans"
)

func convertVulns(dockerVulnDetails []*vulnerabilityDetails) []*v1.Vulnerability {
	vulns := make([]*v1.Vulnerability, len(dockerVulnDetails))
	for i, vulnDetails := range dockerVulnDetails {
		vuln := vulnDetails.Vulnerability
		vulns[i] = &v1.Vulnerability{
			Cve:     vuln.CVE,
			Cvss:    vuln.CVSS,
			Summary: vuln.Summary,
			Link:    scans.GetVulnLink(vuln.CVE),
		}
	}
	return vulns
}

func convertLicense(license *license) *v1.License {
	if license == nil {
		return nil
	}
	return &v1.License{
		Name: license.Name,
		Type: license.Type,
		Url:  license.URL,
	}
}

func convertComponents(dockerComponents []*component) []*v1.ImageScanComponent {
	components := make([]*v1.ImageScanComponent, len(dockerComponents))
	for i, component := range dockerComponents {
		convertedVulns := convertVulns(component.Vulnerabilities)
		components[i] = &v1.ImageScanComponent{
			Name:    component.Component,
			Version: component.Version,
			License: convertLicense(component.License),
			Vulns:   convertedVulns,
		}
	}
	return components
}

func convertLayers(layerDetails []*detailedSummary) []*v1.ImageScanComponent {
	components := make([]*v1.ImageScanComponent, 0, len(layerDetails))
	for _, layerDetail := range layerDetails {
		convertedComponents := convertComponents(layerDetail.Components)
		components = append(components, convertedComponents...)
	}
	return components
}

func compareComponent(c1, c2 *v1.ImageScanComponent) int {
	if c1.GetName() < c2.GetName() {
		return -1
	} else if c1.GetName() > c2.GetName() {
		return 1
	}
	if c1.GetVersion() < c2.GetVersion() {
		return -1
	} else if c1.GetVersion() > c2.GetVersion() {
		return 1
	}
	return 0
}

func convertTagScanSummaryToImageScan(tagScanSummary *tagScanSummary) *v1.ImageScan {
	convertedLayers := convertLayers(tagScanSummary.LayerDetails)
	completedAt, err := ptypes.TimestampProto(tagScanSummary.CheckCompletedAt)
	if err != nil {
		log.Error(err)
	}

	// Deduplicate the components by sorting first then iterating
	sort.SliceStable(convertedLayers, func(i, j int) bool {
		return compareComponent(convertedLayers[i], convertedLayers[j]) <= 0
	})
	uniqueLayers := convertedLayers[:1]
	for i := 1; i < len(convertedLayers); i++ {
		prevComponent, currComponent := convertedLayers[i-1], convertedLayers[i]
		if compareComponent(prevComponent, currComponent) == 0 {
			continue
		}
		uniqueLayers = append(uniqueLayers, currComponent)
	}
	return &v1.ImageScan{
		ScanTime:   completedAt,
		Components: convertedLayers,
	}
}
