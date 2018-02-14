package dtr

import (
	"testing"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
)

func getTestVulns() ([]*vulnerabilityDetails, []*v1.Vulnerability) {
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
				Summary: "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0682, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418.",
			},
		},
	}
	v1Vulns := []*v1.Vulnerability{
		{
			Cve:     "CVE-2016-0682",
			Cvss:    6.9,
			Summary: "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0689, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418.",
		},
		{
			Cve:     "CVE-2016-0689",
			Cvss:    6.9,
			Summary: "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0682, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418.",
		},
	}
	return dockerVulnDetails, v1Vulns
}

func getTestLicense() (*license, *v1.License) {
	dockerLicense := &license{
		Name: "name",
		Type: "copyleft",
		URL:  "url",
	}

	v1License := &v1.License{
		Name: "name",
		Type: "copyleft",
		Url:  "url",
	}
	return dockerLicense, v1License
}

func getTestComponents() ([]*component, []*v1.ImageScanComponents) {
	dockerLicense, v1License := getTestLicense()
	dockerVulns, v1Vulns := getTestVulns()

	dockerComponents := []*component{
		{
			Component:       "berkeleydb",
			Version:         "5.3.28-9",
			License:         dockerLicense,
			Vulnerabilities: dockerVulns,
			FullPath: []string{
				"usr/lib/x86_64-linux-gnu/libdb-5.3.so",
			},
		},
	}
	v1Components := []*v1.ImageScanComponents{
		{
			Name:    "berkeleydb",
			Version: "5.3.28-9",
			License: v1License,
			FullPath: []string{
				"usr/lib/x86_64-linux-gnu/libdb-5.3.so",
			},
			Vulns: v1Vulns,
		},
	}
	return dockerComponents, v1Components
}

func getTestLayers() ([]*detailedSummary, []*v1.ImageScanComponents) {
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
	actualComponents := convertComponents(dockerComponents)
	assert.Equal(t, expectedComponents, actualComponents)
}

func TestConvertLayers(t *testing.T) {
	dockerLayers, expectedLayers := getTestLayers()
	actualLayers := convertLayers(dockerLayers)
	assert.Equal(t, expectedLayers, actualLayers)
}

func TestConvertScanState(t *testing.T) {
	assert.Equal(t, v1.ImageScanState_FAILED, convertScanState(failed))
	assert.Equal(t, v1.ImageScanState_UNSCANNED, convertScanState(unscanned))
	assert.Equal(t, v1.ImageScanState_SCANNING, convertScanState(scanning))
	assert.Equal(t, v1.ImageScanState_PENDING, convertScanState(pending))
	assert.Equal(t, v1.ImageScanState_SCANNED, convertScanState(scanned))
	assert.Equal(t, v1.ImageScanState_CHECKING, convertScanState(checking))
	assert.Equal(t, v1.ImageScanState_COMPLETED, convertScanState(completed))
}

func TestConvertTagScanSummariesToImageScans(t *testing.T) {
	dockerLayers, expectedComponents := getTestLayers()
	tagScanSummaries := []*tagScanSummary{
		{
			Namespace:        "docker",
			RepoName:         "nginx",
			Tag:              "latest",
			LastScanStatus:   checking,
			LayerDetails:     dockerLayers,
			CheckCompletedAt: time.Unix(0, 1000),
		},
	}
	protoTime, _ := ptypes.TimestampProto(time.Unix(0, 1000))
	expectedScans := []*v1.ImageScan{
		{
			Name: &v1.ImageName{
				Registry: "registry",
				Remote:   "docker/nginx",
				Tag:      "latest",
			},
			State:      v1.ImageScanState_CHECKING,
			Components: expectedComponents,
			ScanTime:   protoTime,
		},
	}

	actualScans := convertTagScanSummariesToImageScans("registry", tagScanSummaries)
	assert.Equal(t, expectedScans, actualScans)
}
