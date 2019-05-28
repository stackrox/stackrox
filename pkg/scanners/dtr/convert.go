package dtr

import (
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
)

func convertVulns(dockerVulnDetails []*vulnerabilityDetails) []*storage.Vulnerability {
	vulns := make([]*storage.Vulnerability, len(dockerVulnDetails))
	for i, vulnDetails := range dockerVulnDetails {
		vuln := vulnDetails.Vulnerability
		vulns[i] = &storage.Vulnerability{
			Cve:     vuln.CVE,
			Cvss:    vuln.CVSS,
			Summary: stringutils.Truncate(vuln.Summary, 64, stringutils.WordOriented{}),
			Link:    scans.GetVulnLink(vuln.CVE),
		}
	}
	return vulns
}

func convertLicense(license *license) *storage.License {
	if license == nil {
		return nil
	}
	return &storage.License{
		Name: license.Name,
		Type: license.Type,
		Url:  license.URL,
	}
}

func convertComponents(dockerComponents []*component) []*storage.ImageScanComponent {
	components := make([]*storage.ImageScanComponent, len(dockerComponents))
	for i, component := range dockerComponents {
		convertedVulns := convertVulns(component.Vulnerabilities)
		components[i] = &storage.ImageScanComponent{
			Name:    component.Component,
			Version: component.Version,
			License: convertLicense(component.License),
			Vulns:   convertedVulns,
		}
	}
	return components
}

func convertLayers(layerDetails []*detailedSummary) []*storage.ImageScanComponent {
	components := make([]*storage.ImageScanComponent, 0, len(layerDetails))
	for _, layerDetail := range layerDetails {
		convertedComponents := convertComponents(layerDetail.Components)
		components = append(components, convertedComponents...)
	}
	return components
}

func compareComponent(c1, c2 *storage.ImageScanComponent) int {
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

func populateLayersWithScan(image *storage.Image, layerDetails []*detailedSummary) {
	layerDetailIdx := 0
	for _, l := range image.GetMetadata().GetV1().GetLayers() {
		if !l.Empty {
			l.Components = convertComponents(layerDetails[layerDetailIdx].Components)
			layerDetailIdx++
		}
	}
}

func convertTagScanSummaryToImageScan(tagScanSummary *tagScanSummary) *storage.ImageScan {
	convertedLayers := convertLayers(tagScanSummary.LayerDetails)
	completedAt, err := ptypes.TimestampProto(tagScanSummary.CheckCompletedAt)
	if err != nil {
		log.Error(err)
	}

	// Deduplicate the components by sorting first then iterating
	sort.SliceStable(convertedLayers, func(i, j int) bool {
		return compareComponent(convertedLayers[i], convertedLayers[j]) <= 0
	})

	if len(convertedLayers) == 0 {
		return &storage.ImageScan{
			ScanTime: completedAt,
		}
	}

	uniqueLayers := convertedLayers[:1]
	for i := 1; i < len(convertedLayers); i++ {
		prevComponent, currComponent := convertedLayers[i-1], convertedLayers[i]
		if compareComponent(prevComponent, currComponent) == 0 {
			continue
		}
		uniqueLayers = append(uniqueLayers, currComponent)
	}

	return &storage.ImageScan{
		ScanTime:   completedAt,
		Components: uniqueLayers,
	}
}
