package dtr

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
)

func convertVulns(dockerVulnDetails []*vulnerabilityDetails) []*v1.Vulnerability {
	vulns := make([]*v1.Vulnerability, len(dockerVulnDetails))
	for i, vulnDetails := range dockerVulnDetails {
		vuln := vulnDetails.Vulnerability
		vulns[i] = &v1.Vulnerability{
			Cve:     vuln.CVE,
			Cvss:    vuln.CVSS,
			Summary: vuln.Summary,
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

func convertComponents(dockerComponents []*component) []*v1.ImageScanComponents {
	components := make([]*v1.ImageScanComponents, len(dockerComponents))
	for i, component := range dockerComponents {
		convertedVulns := convertVulns(component.Vulnerabilities)
		components[i] = &v1.ImageScanComponents{
			Name:     component.Component,
			Version:  component.Version,
			License:  convertLicense(component.License),
			FullPath: component.FullPath,
			Vulns:    convertedVulns,
		}
	}
	return components
}

func convertLayers(layerDetails []*detailedSummary) []*v1.ScanLayer {
	layers := make([]*v1.ScanLayer, len(layerDetails))
	for i, layerDetail := range layerDetails {
		convertedComponents := convertComponents(layerDetail.Components)
		layers[i] = &v1.ScanLayer{
			Sha:        layerDetail.SHA256Sum,
			Components: convertedComponents,
		}
	}
	return layers
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
			Registry: server,
			Remote:   fmt.Sprintf("%v/%v", tagScan.Namespace, tagScan.RepoName),
			Tag:      tagScan.Tag,
			State:    convertScanState(tagScan.LastScanStatus),
			Layers:   convertedLayers,
			ScanTime: completedAt,
		})
	}
	return imageScans
}

func convertScanState(status scanStatus) v1.ImageScanState {
	upper := strings.ToUpper(status.String())
	val := v1.ImageScanState_value[upper]
	return v1.ImageScanState(val)
}
