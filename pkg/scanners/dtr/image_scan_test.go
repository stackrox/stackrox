package dtr

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const scanResultPayload = `[
  {
   "namespace": "docker",
   "reponame": "nginx",
   "tag": "1.10",
   "critical": 8,
   "major": 25,
   "minor": 3,
   "last_scan_status": 6,
   "check_completed_at": "2017-11-16T21:39:07.676Z",
   "should_rescan": true,
   "has_foreign_layers": true,
   "layer_details": [
    {
     "sha256sum": "ef24d3d19d383c557b3bb92c21cc1b3e0c4ca6735160b6d3c684fb92ba0b3569",
     "components": [
      {
       "component": "berkeleydb",
       "version": "5.3.28-9",
       "license": {
        "name": "sleepycat",
        "type": "copyleft",
        "url": "https://opensource.org/licenses/Sleepycat"
       },
       "vulns": [
        {
         "vuln": {
          "cve": "CVE-2016-0682",
          "cvss": 6.9,
          "summary": "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0689, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418."
         },
         "exact": true,
         "notes": null
        },
        {
         "vuln": {
          "cve": "CVE-2016-0689",
          "cvss": 6.9,
          "summary": "Unspecified vulnerability in the DataStore component in Oracle Berkeley DB 11.2.5.0.32, 11.2.5.1.29, 11.2.5.2.42, 11.2.5.3.28, 12.1.6.0.35, and 12.1.6.1.26 allows local users to affect confidentiality, integrity, and availability via unknown vectors, a different vulnerability than CVE-2016-0682, CVE-2016-0692, CVE-2016-0694, and CVE-2016-3418."
         },
         "exact": true,
         "notes": null
        }
       ],
       "fullpath": [
        "usr/lib/x86_64-linux-gnu/libdb-5.3.so"
       ]
      }
     ]
    }
   ]
  }
 ]
`

func getExpectedImageScan() (*tagScanSummary, error) {
	complete, err := time.Parse(time.RFC3339, "2017-11-16T21:39:07.676Z")
	if err != nil {
		return nil, err
	}
	return &tagScanSummary{
		CheckCompletedAt: complete,
		LayerDetails: []*detailedSummary{
			{
				SHA256Sum: "ef24d3d19d383c557b3bb92c21cc1b3e0c4ca6735160b6d3c684fb92ba0b3569",
				Components: []*component{
					{
						Component: "berkeleydb",
						Version:   "5.3.28-9",
						License: &license{
							Name: "sleepycat",
							Type: "copyleft",
							URL:  "https://opensource.org/licenses/Sleepycat",
						},
						Vulnerabilities: []*vulnerabilityDetails{
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
						},
					},
				},
			},
		},
	}, nil
}

func TestParseImageScan(t *testing.T) {
	actualScans, err := parseDTRImageScans([]byte(scanResultPayload))
	require.Nil(t, err)

	expectedScan, err := getExpectedImageScan()
	assert.Nil(t, err)
	assert.Equal(t, expectedScan, actualScans[0])
}
