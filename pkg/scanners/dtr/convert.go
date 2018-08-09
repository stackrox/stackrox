package dtr

import (
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

func convertTagScanSummariesToImageScans(server string, tagScanSummaries []*tagScanSummary) []*v1.ImageScan {
	imageScans := make([]*v1.ImageScan, 0, len(tagScanSummaries))
	for _, tagScan := range tagScanSummaries {
		convertedLayers := convertLayers(tagScan.LayerDetails)

		completedAt, err := ptypes.TimestampProto(tagScan.CheckCompletedAt)
		if err != nil {
			log.Error(err)
		}

		imageScans = append(imageScans, &v1.ImageScan{
			ScanTime:   completedAt,
			Components: convertedLayers,
		})
	}
	return imageScans
}
