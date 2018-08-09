package tenable

import (
	"sort"
	"testing"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/scans"
	"github.com/stretchr/testify/assert"
)

func getFindingsAndPackages() ([]*finding, []pkg, []*v1.ImageScanComponent) {
	findings := []*finding{
		{
			NVDFinding: nvdFinding{
				ReferenceID:           "DSA-3566",
				CVE:                   "CVE-2016-2109",
				PublishedDate:         "2016/05/03",
				ModifiedDate:          "2016/05/03",
				Description:           "CVE Description",
				CVSSScore:             "10.0",
				AccessVector:          "Network",
				AccessComplexity:      "Low",
				Auth:                  "None required",
				AvailabilityImpact:    "Complete",
				ConfidentialityImpact: "Complete",
				IntegrityImpact:       "Complete",
				CWE:                   "",
				CPE: []string{
					"p-cpe:/a:debian:debian_linux:openssl",
				},
				Remediation: "Upgrade the openssl packages.\n\nFor the stable distribution (jessie), these problems have been fixed\nin version 1.0.1k-3+deb8u5.",
				References: []string{
					"DSA:3566",
				},
			},
			Packages: []pkg{
				{
					Name:    "libssl1.0.0",
					Version: "1.0.1t-1+deb8u6",
				},
				{
					Name:    "openssl",
					Version: "1.0.1t-1+deb8u6",
				},
			},
		},
		{
			NVDFinding: nvdFinding{
				ReferenceID:           "DSA-3903",
				CVE:                   "CVE-2017-9936",
				PublishedDate:         "2017/07/05",
				ModifiedDate:          "2017/07/05",
				Description:           "Description 2",
				CVSSScore:             "5.0",
				AccessVector:          "Network",
				AccessComplexity:      "Low",
				Auth:                  "None required",
				AvailabilityImpact:    "Partial",
				ConfidentialityImpact: "None",
				IntegrityImpact:       "None",
				CWE:                   "",
				CPE: []string{
					"p-cpe:/a:debian:debian_linux:tiff",
				},
				Remediation: "Upgrade the tiff packages.\n\nFor the oldstable distribution (jessie), these problems have been\nfixed in version 4.0.3-12.3+deb8u4.\n\nFor the stable distribution (stretch), these problems have been fixed\nin version 4.0.8-2+deb9u1.",
				References: []string{
					"DSA:3903",
				},
			},
			Packages: []pkg{
				{
					Name:    "libtiff5",
					Version: "4.0.3-12.3+deb8u2",
				},
			},
		},
	}
	packages := []pkg{
		{
			Name:    "libtiff5",
			Version: "4.0.3-12.3+deb8u2",
		},
		{
			Name:    "openssl",
			Version: "1.0.1t-1+deb8u6",
		},
		{
			Name:    "debianutils",
			Version: "4.4+b1",
		},
		{
			Name:    "libssl1.0.0",
			Version: "1.0.1t-1+deb8u6",
		},
	}

	components := []*v1.ImageScanComponent{
		{
			Name:    "libssl1.0.0",
			Version: "1.0.1t-1+deb8u6",
			Vulns: []*v1.Vulnerability{
				{
					Cve:     "CVE-2016-2109",
					Cvss:    10.0,
					Summary: "CVE Description",
					Link:    scans.GetVulnLink("CVE-2016-2109"),
				},
			},
		},
		{
			Name:    "openssl",
			Version: "1.0.1t-1+deb8u6",
			Vulns: []*v1.Vulnerability{
				{
					Cve:     "CVE-2016-2109",
					Cvss:    10.0,
					Summary: "CVE Description",
					Link:    scans.GetVulnLink("CVE-2016-2109"),
				},
			},
		},
		{
			Name:    "libtiff5",
			Version: "4.0.3-12.3+deb8u2",
			Vulns: []*v1.Vulnerability{
				{
					Cve:     "CVE-2017-9936",
					Cvss:    5.0,
					Summary: "Description 2",
					Link:    scans.GetVulnLink("CVE-2017-9936"),
				},
			},
		},
		{
			Name:    "debianutils",
			Version: "4.4+b1",
		},
	}
	return findings, packages, components
}

func sortComponents(c []*v1.ImageScanComponent) {
	sort.SliceStable(c, func(i, j int) bool { return c[i].Name < c[j].Name })
}

func TestConvertNVDFindingsAndPackagesToComponents(t *testing.T) {
	findings, packages, expectedComponents := getFindingsAndPackages()
	convertedComponents := convertNVDFindingsAndPackagesToComponents(findings, packages)
	// There is no ordering constraint on components as they are converted using a map so sort first and then compare
	sort.SliceStable(expectedComponents, func(i, j int) bool { return expectedComponents[i].Name < expectedComponents[j].Name })
	sort.SliceStable(convertedComponents, func(i, j int) bool { return convertedComponents[i].Name < convertedComponents[j].Name })

	assert.Equal(t, expectedComponents, convertedComponents)
}

func TestConvertScanToImageScan(t *testing.T) {
	findings, packages, components := getFindingsAndPackages()

	created := time.Now()
	updated := time.Now().AddDate(0, 0, 1)

	scan := &scanResult{
		ID:                "6984854121115593873",
		ImageName:         "nginx",
		DockerImageID:     "0346349a1a64",
		Tag:               "1.10",
		CreatedAt:         created,
		UpdatedAt:         updated,
		Platform:          "docker",
		OSArch:            "AMD64",
		OS:                "LINUX_DEBIAN",
		SHA256:            "sha256:56eefbfef9aa918410e5cfb97a1e83a52d7ac3989ca9e4fe8baa9db8156372bd",
		OSVersion:         "8.7",
		RiskScore:         6.0,
		Digest:            "56eefbfef9aa918410e5cfb97a1e83a52d7ac3989ca9e4fe8baa9db8156372bd",
		InstalledPackages: packages,
		Findings:          findings,
	}

	image := &v1.Image{
		Name: &v1.ImageName{
			Registry: "",
			Remote:   "srox/nginx",
			Tag:      "1.10",
		},
	}

	scanTime, err := ptypes.TimestampProto(updated)
	assert.NoError(t, err)
	expectedScan := &v1.ImageScan{
		Components: components,
		ScanTime:   scanTime,
	}

	convertedScan := convertScanToImageScan(image, scan)
	sortComponents(convertedScan.Components)
	sortComponents(expectedScan.Components)
	assert.Equal(t, expectedScan, convertedScan)
}
