package quay

import (
	"testing"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scanPayload = `
{
  "status": "scanned",
  "data": {
    "Layer": {
      "Features": [
        {
          "Version": "1.10.3-1~jessie",
          "Name": "nginx-module-geoip"
        },
        {
          "Name": "pcre3",
          "Version": "2:8.35-3.3+deb8u4",
          "Vulnerabilities": [
            {
              "Link": "https://security-tracker.debian.org/tracker/CVE-2017-16231",
              "Name": "CVE-2017-16231"
            },
            {
              "Link": "https://security-tracker.debian.org/tracker/CVE-2017-7246",
              "Description": "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
              "Name": "CVE-2017-7246",
              "Metadata": {
                "NVD": {
                  "CVSSv2": {
                    "Score": 6.8,
					"Vectors": "AV:L/AC:L/Au:N/C:N/I:P"
                  }
                }
              }
            }
          ]
        }
      ]
    }
  }
}
`

func getImageScan() (*scanResult, error) {
	return &scanResult{
		Status: "scanned",
		Data: clairV1.LayerEnvelope{
			Layer: &clairV1.Layer{
				Features: []clairV1.Feature{
					{
						Name:    "nginx-module-geoip",
						Version: "1.10.3-1~jessie",
					},
					{
						Name:    "pcre3",
						Version: "2:8.35-3.3+deb8u4",
						Vulnerabilities: []clairV1.Vulnerability{
							{
								Link: "https://security-tracker.debian.org/tracker/CVE-2017-16231",
								Name: "CVE-2017-16231",
							},
							{
								Link:        "https://security-tracker.debian.org/tracker/CVE-2017-7246",
								Description: "Stack-based buffer overflow in the pcre32_copy_substring function in pcre_get.c in libpcre1 in PCRE 8.40 allows remote attackers to cause a denial of service (WRITE of size 268) or possibly have unspecified other impact via a crafted file.",
								Name:        "CVE-2017-7246",
								Metadata: map[string]interface{}{
									"NVD": map[string]interface{}{
										"CVSSv2": map[string]interface{}{
											"Score":   6.8,
											"Vectors": "AV:L/AC:L/Au:N/C:N/I:P",
										},
									},
								},
							},
						},
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
	assert.Equal(t, expectedScan, actualScan)
}
