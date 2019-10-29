package dtr

import (
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stackrox/rox/pkg/stringutils"
)

func convertVulns(dockerVulnDetails []*vulnerabilityDetails) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, len(dockerVulnDetails))
	for i, vulnDetails := range dockerVulnDetails {
		vuln := vulnDetails.Vulnerability
		vulns[i] = &storage.EmbeddedVulnerability{
			Cve:     vuln.CVE,
			Cvss:    vuln.CVSS,
			Summary: stringutils.TruncateIf(vuln.Summary, 64, !features.VulnMgmtUI.Enabled(), stringutils.WordOriented{}),
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

func convertComponents(layerIdx *int32, dockerComponents []*component) []*storage.EmbeddedImageScanComponent {
	components := make([]*storage.EmbeddedImageScanComponent, len(dockerComponents))
	for i, component := range dockerComponents {
		convertedVulns := convertVulns(component.Vulnerabilities)
		components[i] = &storage.EmbeddedImageScanComponent{
			Name:    component.Component,
			Version: component.Version,
			License: convertLicense(component.License),
			Vulns:   convertedVulns,
		}
		if layerIdx != nil {
			components[i].HasLayerIndex = &storage.EmbeddedImageScanComponent_LayerIndex{
				LayerIndex: *layerIdx,
			}
		}
	}
	return components
}

func convertLayers(image *storage.Image, layerDetails []*detailedSummary) []*storage.EmbeddedImageScanComponent {
	var nonEmptyLayers []int32
	for i, l := range image.GetMetadata().GetV1().GetLayers() {
		if !l.GetEmpty() {
			nonEmptyLayers = append(nonEmptyLayers, int32(i))
		}
	}
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(layerDetails))
	for i, layerDetail := range layerDetails {
		var layerIdx *int32
		if i >= len(nonEmptyLayers) {
			log.Error("Received unexpected number of layer details")
		} else {
			layerIdx = &nonEmptyLayers[i]
		}
		convertedComponents := convertComponents(layerIdx, layerDetail.Components)
		components = append(components, convertedComponents...)
	}
	return components
}

func compareComponent(c1, c2 *storage.EmbeddedImageScanComponent) int {
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

func convertTagScanSummaryToImageScan(image *storage.Image, tagScanSummary *tagScanSummary) *storage.ImageScan {
	convertedLayers := convertLayers(image, tagScanSummary.LayerDetails)
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
