package tenable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scanPayload = `
{
  "id": "6984854121115593873",
  "image_name": "nginx",
  "docker_image_id": "0346349a1a64",
  "tag": "1.10",
  "created_at": "2017-12-21T06:18:43.634Z",
  "updated_at": "2017-12-21T06:18:43.634Z",
  "platform": "docker",
  "findings": [
    {
      "nvdFinding": {
        "reference_id": "DSA-3566",
        "cve": "CVE-2016-2109",
        "published_date": "2016/05/03",
        "modified_date": "2016/05/03",
        "description": "CVE Description",
        "cvss_score": "10.0",
        "access_vector": "Network",
        "access_complexity": "Low",
        "auth": "None required",
        "availability_impact": "Complete",
        "confidentiality_impact": "Complete",
        "integrity_impact": "Complete",
        "cwe": "",
        "cpe": [
          "p-cpe:/a:debian:debian_linux:openssl"
        ],
        "remediation": "Upgrade the openssl packages.\n\nFor the stable distribution (jessie), these problems have been fixed\nin version 1.0.1k-3+deb8u5.",
        "references": [
          "DSA:3566"
        ]
      },
      "packages": [
        {
          "name": "libssl1.0.0",
          "version": "1.0.1t-1+deb8u6"
        },
        {
          "name": "openssl",
          "version": "1.0.1t-1+deb8u6"
        }
      ]
    },
    {
      "nvdFinding": {
        "reference_id": "DSA-3903",
        "cve": "CVE-2017-9936",
        "published_date": "2017/07/05",
        "modified_date": "2017/07/05",
        "description": "Description 2",
        "cvss_score": "5.0",
        "access_vector": "Network",
        "access_complexity": "Low",
        "auth": "None required",
        "availability_impact": "Partial",
        "confidentiality_impact": "None",
        "integrity_impact": "None",
        "cwe": "",
        "cpe": [
          "p-cpe:/a:debian:debian_linux:tiff"
        ],
        "remediation": "Upgrade the tiff packages.\n\nFor the oldstable distribution (jessie), these problems have been\nfixed in version 4.0.3-12.3+deb8u4.\n\nFor the stable distribution (stretch), these problems have been fixed\nin version 4.0.8-2+deb9u1.",
        "references": [
          "DSA:3903"
        ]
      },
      "packages": [
        {
          "name": "libtiff5",
          "version": "4.0.3-12.3+deb8u2"
        }
      ]
    }
  ],
  "malware": [],
  "potentially_unwanted_programs": [],
  "os_architecture": "AMD64",
  "sha256": "sha256:56eefbfef9aa918410e5cfb97a1e83a52d7ac3989ca9e4fe8baa9db8156372bd",
  "os": "LINUX_DEBIAN",
  "os_version": "8.7",
  "installed_packages": [
    {
      "name": "libtiff5",
      "version": "4.0.3-12.3+deb8u2"
    },
    {
      "name": "openssl",
      "version": "1.0.1t-1+deb8u6"
    },
    {
      "name": "debianutils",
      "version": "4.4+b1"
    },
    {
      "name": "libssl1.0.0",
      "version": "1.0.1t-1+deb8u6"
    }
  ],
  "risk_score": 6,
  "digest": "56eefbfef9aa918410e5cfb97a1e83a52d7ac3989ca9e4fe8baa9db8156372bd"
}
`

func getImageScan() (*scanResult, error) {
	createdAt, err := time.Parse(time.RFC3339, "2017-12-21T06:18:43.634Z")
	if err != nil {
		return nil, err
	}
	return &scanResult{
		ID:                          "6984854121115593873",
		ImageName:                   "nginx",
		DockerImageID:               "0346349a1a64",
		Tag:                         "1.10",
		CreatedAt:                   createdAt,
		UpdatedAt:                   createdAt,
		Platform:                    "docker",
		OSArch:                      "AMD64",
		OS:                          "LINUX_DEBIAN",
		SHA256:                      "sha256:56eefbfef9aa918410e5cfb97a1e83a52d7ac3989ca9e4fe8baa9db8156372bd",
		OSVersion:                   "8.7",
		RiskScore:                   6.0,
		Digest:                      "56eefbfef9aa918410e5cfb97a1e83a52d7ac3989ca9e4fe8baa9db8156372bd",
		Malware:                     []interface{}{},
		PotentiallyUnwantedPrograms: []interface{}{},
		InstalledPackages: []pkg{
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
		},
		Findings: []*finding{
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
		},
	}, nil
}

func TestParseImageScan(t *testing.T) {
	actualScan, err := parseImageScan([]byte(scanPayload))
	require.Nil(t, err)

	expectedScan, err := getImageScan()
	assert.Nil(t, err)

	// The wall part of the time does not match
	assert.Equal(t, expectedScan.CreatedAt.String(), actualScan.CreatedAt.String())
	assert.Equal(t, expectedScan.UpdatedAt.String(), actualScan.UpdatedAt.String())

	ti := time.Now()
	expectedScan.CreatedAt = ti
	expectedScan.UpdatedAt = ti
	actualScan.CreatedAt = ti
	actualScan.UpdatedAt = ti
	assert.Equal(t, expectedScan, actualScan)
}
