package dtr

import (
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stretchr/testify/assert"
)

func getTestVulns() ([]*vulnerabilityDetails, []*storage.Vulnerability) {
	dockerVulnDetails := []*vulnerabilityDetails{
		{
			Vulnerability: &vulnerability{
				CVE:     "CVE-2016-0682",
				CVSS:    6.9,
				Summary: "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0689, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418.",
			},
		},
		{
			Vulnerability: &vulnerability{
				CVE:     "CVE-2016-0689",
				CVSS:    6.9,
				Summary: "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0689, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418.",
			},
		},
	}
	v1Vulns := []*storage.Vulnerability{
		{
			Cve:     "CVE-2016-0682",
			Cvss:    6.9,
			Summary: "Unspecified vulnerability in the DataStore component in Oracle...",
			Link:    scans.GetVulnLink("CVE-2016-0682"),
		},
		{
			Cve:     "CVE-2016-0689",
			Cvss:    6.9,
			Summary: "Unspecified vulnerability in the DataStore component in Oracle...",
			Link:    scans.GetVulnLink("CVE-2016-0689"),
		},
	}
	return dockerVulnDetails, v1Vulns
}

func getTestLicense() (*license, *storage.License) {
	dockerLicense := &license{
		Name: "name",
		Type: "copyleft",
		URL:  "url",
	}

	v1License := &storage.License{
		Name: "name",
		Type: "copyleft",
		Url:  "url",
	}
	return dockerLicense, v1License
}

func getTestComponents() ([]*component, []*storage.ImageScanComponent) {
	dockerLicense, v1License := getTestLicense()
	dockerVulns, v1Vulns := getTestVulns()

	dockerComponents := []*component{
		{
			Component:       "berkeleydb",
			Version:         "5.3.28-9",
			License:         dockerLicense,
			Vulnerabilities: dockerVulns,
		},
	}
	v1Components := []*storage.ImageScanComponent{
		{
			Name:    "berkeleydb",
			Version: "5.3.28-9",
			License: v1License,
			Vulns:   v1Vulns,
		},
	}
	return dockerComponents, v1Components
}

func getTestLayers() ([]*detailedSummary, []*storage.ImageScanComponent) {
	dockerComponents, v1Components := getTestComponents()

	dockerLayers := []*detailedSummary{
		{
			SHA256Sum:  "sha",
			Components: dockerComponents,
		},
	}
	return dockerLayers, v1Components
}

func TestConvertVulns(t *testing.T) {
	dockerVulnDetails, expectedVulns := getTestVulns()
	actualVulns := convertVulns(dockerVulnDetails)
	assert.Equal(t, expectedVulns, actualVulns)
}

func TestConvertLicense(t *testing.T) {
	assert.Nil(t, convertLicense(nil))
	dockerLicense, expectedLicense := getTestLicense()
	actualLicense := convertLicense(dockerLicense)
	assert.Equal(t, expectedLicense, actualLicense)
}

func TestConvertComponents(t *testing.T) {
	dockerComponents, expectedComponents := getTestComponents()
	actualComponents := convertComponents(nil, dockerComponents)
	assert.Equal(t, expectedComponents, actualComponents)
}

func TestConvertLayers(t *testing.T) {
	dockerLayers, expectedLayers := getTestLayers()
	actualLayers := convertLayers(nil, dockerLayers)
	assert.Equal(t, expectedLayers, actualLayers)
}

func TestConvertTagScanSummariesToImageScans(t *testing.T) {
	dockerLayers, expectedComponents := getTestLayers()
	tagScanSummary := &tagScanSummary{
		LayerDetails:     dockerLayers,
		CheckCompletedAt: time.Unix(0, 1000),
	}
	protoTime, _ := ptypes.TimestampProto(time.Unix(0, 1000))
	expectedScan := &storage.ImageScan{
		Components: expectedComponents,
		ScanTime:   protoTime,
	}

	actualScans := convertTagScanSummaryToImageScan(nil, tagScanSummary)
	assert.Equal(t, expectedScan, actualScans)
}
