package mappers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractNVDVulnerabilities(t *testing.T) {
	enrichments := map[string][]json.RawMessage{
		"message/vnd.clair.map.vulnerability; enricher=nvd": {
			json.RawMessage(`{
				"CVE-2021-1234": [{
					"cve": "CVE-2021-1234",
					"cvss": [{
						"version": "3.1",
						"score": 7.5,
						"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"
					}]
				}]
			}`),
			json.RawMessage(`{
				"CVE-2021-5678": [{
					"cve": "CVE-2021-5678",
					"cvss": [{
						"version": "2.0",
						"score": 5.0,
						"vector": "AV:N/AC:L/Au:N/C:N/I:N/A:P"
					}, {
						"version": "3.1",
						"score": 9.8,
						"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
					}]
				}]
			}`),
		},
		"message/vnd.clair.map.vulnerability; enricher=other": {
			json.RawMessage(`{
				"CVE-2022-9999": [{
					"cve": "CVE-2022-9999",
					"cvss": [{
						"version": "3.0",
						"score": 4.3,
						"vector": "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N"
					}]
				}]
			}`),
		},
	}

	result, err := ExtractNVDVulnerabilities(enrichments)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check CVE-2021-1234 with only CVSSv3
	nvd1234 := result["message/vnd.clair.map.vulnerability; enricher=nvd"]["CVE-2021-1234"]
	require.NotNil(t, nvd1234)
	assert.Nil(t, nvd1234.CVSSv2)
	require.NotNil(t, nvd1234.CVSSv3)
	assert.Equal(t, 7.5, nvd1234.CVSSv3.BaseScore)
	assert.Equal(t, "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", nvd1234.CVSSv3.Vector)

	// Check CVE-2021-5678 with both CVSSv2 and CVSSv3
	nvd5678 := result["message/vnd.clair.map.vulnerability; enricher=nvd"]["CVE-2021-5678"]
	require.NotNil(t, nvd5678)
	require.NotNil(t, nvd5678.CVSSv2)
	assert.Equal(t, 5.0, nvd5678.CVSSv2.BaseScore)
	assert.Equal(t, "AV:N/AC:L/Au:N/C:N/I:N/A:P", nvd5678.CVSSv2.Vector)
	require.NotNil(t, nvd5678.CVSSv3)
	assert.Equal(t, 9.8, nvd5678.CVSSv3.BaseScore)
	assert.Equal(t, "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", nvd5678.CVSSv3.Vector)

	// Check CVE-2022-9999 from other enricher
	nvd9999 := result["message/vnd.clair.map.vulnerability; enricher=other"]["CVE-2022-9999"]
	require.NotNil(t, nvd9999)
	assert.Nil(t, nvd9999.CVSSv2)
	require.NotNil(t, nvd9999.CVSSv3)
	assert.Equal(t, 4.3, nvd9999.CVSSv3.BaseScore)
	assert.Equal(t, "CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:L/I:N/A:N", nvd9999.CVSSv3.Vector)
}

func TestExtractNVDVulnerabilities_NoEnrichments(t *testing.T) {
	testCases := map[string]struct {
		input    map[string][]json.RawMessage
		expected map[string]map[string]*NVDItem
	}{
		"nil input": {
			input:    nil,
			expected: map[string]map[string]*NVDItem{},
		},
		"empty input": {
			input:    map[string][]json.RawMessage{},
			expected: map[string]map[string]*NVDItem{},
		},
		"no matching keys": {
			input: map[string][]json.RawMessage{
				"other/key": {json.RawMessage(`{}`)},
			},
			expected: map[string]map[string]*NVDItem{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := ExtractNVDVulnerabilities(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractEPSS(t *testing.T) {
	enrichments := map[string][]json.RawMessage{
		"message/vnd.clair.map.enrichment; enricher=epss": {
			json.RawMessage(`{
				"CVE-2021-1234": [{
					"cve": "CVE-2021-1234",
					"epss": {
						"model_version": "v2023.03.01",
						"date": "2023-03-15",
						"probability": 0.75432,
						"percentile": 0.92156
					}
				}]
			}`),
			json.RawMessage(`{
				"CVE-2021-5678": [{
					"cve": "CVE-2021-5678",
					"epss": {
						"model_version": "v2023.03.01",
						"date": "2023-03-15",
						"probability": 0.00123,
						"percentile": 0.12345
					}
				}]
			}`),
		},
	}

	result, err := ExtractEPSS(enrichments)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check CVE-2021-1234
	epss1234 := result["message/vnd.clair.map.enrichment; enricher=epss"]["CVE-2021-1234"]
	require.NotNil(t, epss1234)
	assert.Equal(t, "v2023.03.01", epss1234.ModelVersion)
	assert.Equal(t, "2023-03-15", epss1234.Date)
	assert.Equal(t, 0.75432, epss1234.Probability)
	assert.Equal(t, 0.92156, epss1234.Percentile)

	// Check CVE-2021-5678
	epss5678 := result["message/vnd.clair.map.enrichment; enricher=epss"]["CVE-2021-5678"]
	require.NotNil(t, epss5678)
	assert.Equal(t, "v2023.03.01", epss5678.ModelVersion)
	assert.Equal(t, "2023-03-15", epss5678.Date)
	assert.Equal(t, 0.00123, epss5678.Probability)
	assert.Equal(t, 0.12345, epss5678.Percentile)
}

func TestExtractCSAFAdvisories(t *testing.T) {
	enrichments := map[string][]json.RawMessage{
		"message/vnd.stackrox.scannerv4.map.csaf; enricher=stackrox.rhel-csaf": {
			json.RawMessage(`{
				"CVE-2021-1234": [{
					"name": "RHSA-2021:1234",
					"description": "Important security update for package",
					"released": "2021-04-15T00:00:00Z",
					"severity": "Important",
					"cvss_v3": {
						"base_score": 7.5,
						"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H"
					}
				}]
			}`),
			json.RawMessage(`{
				"CVE-2021-5678": [{
					"name": "RHSA-2021:5678",
					"description": "Critical security update for library",
					"released": "2021-12-01T00:00:00Z",
					"severity": "Critical",
					"cvss_v3": {
						"base_score": 9.8,
						"vector": "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H"
					},
					"cvss_v2": {
						"base_score": 10.0,
						"vector": "AV:N/AC:L/Au:N/C:C/I:C/A:C"
					}
				}, {
					"name": "RHSA-2021:5679",
					"description": "Another advisory (should be ignored)",
					"released": "2021-12-02T00:00:00Z",
					"severity": "Moderate"
				}]
			}`),
		},
	}

	result, err := ExtractCSAFAdvisories(enrichments)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Check CVE-2021-1234 - first advisory taken
	advisory1234 := result["CVE-2021-1234"]
	require.NotNil(t, advisory1234)
	assert.Equal(t, "RHSA-2021:1234", advisory1234.Name)
	assert.Equal(t, "Important security update for package", advisory1234.Description)
	assert.Equal(t, "Important", advisory1234.Severity)
	expectedDate1 := time.Date(2021, 4, 15, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDate1, advisory1234.ReleaseDate)
	assert.Equal(t, 7.5, advisory1234.CVSSv3.BaseScore)
	assert.Equal(t, "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H", advisory1234.CVSSv3.Vector)
	assert.Equal(t, 0.0, advisory1234.CVSSv2.BaseScore)
	assert.Equal(t, "", advisory1234.CVSSv2.Vector)

	// Check CVE-2021-5678 - first advisory taken, second ignored
	advisory5678 := result["CVE-2021-5678"]
	require.NotNil(t, advisory5678)
	assert.Equal(t, "RHSA-2021:5678", advisory5678.Name)
	assert.Equal(t, "Critical security update for library", advisory5678.Description)
	assert.Equal(t, "Critical", advisory5678.Severity)
	expectedDate2 := time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedDate2, advisory5678.ReleaseDate)
	assert.Equal(t, 9.8, advisory5678.CVSSv3.BaseScore)
	assert.Equal(t, "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", advisory5678.CVSSv3.Vector)
	assert.Equal(t, 10.0, advisory5678.CVSSv2.BaseScore)
	assert.Equal(t, "AV:N/AC:L/Au:N/C:C/I:C/A:C", advisory5678.CVSSv2.Vector)
}

func TestExtractPkgFixedBy(t *testing.T) {
	enrichments := map[string][]json.RawMessage{
		"message/vnd.stackrox.scannerv4.fixedby; enricher=fixedby": {
			json.RawMessage(`{
				"pkg:1": "1.2.3-4.el8",
				"pkg:2": "2.0.0",
				"pkg:3": ""
			}`),
		},
	}

	result, err := ExtractPkgFixedBy(enrichments)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1.2.3-4.el8", result["pkg:1"])
	assert.Equal(t, "2.0.0", result["pkg:2"])
	assert.Equal(t, "", result["pkg:3"])
}

func TestExtractPkgFixedBy_NoEnrichments(t *testing.T) {
	testCases := map[string]struct {
		input    map[string][]json.RawMessage
		expected map[string]string
	}{
		"nil input": {
			input:    nil,
			expected: map[string]string{},
		},
		"empty input": {
			input:    map[string][]json.RawMessage{},
			expected: map[string]string{},
		},
		"no matching key": {
			input: map[string][]json.RawMessage{
				"other/key": {json.RawMessage(`{"pkg:1": "1.0.0"}`)},
			},
			expected: map[string]string{},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result, err := ExtractPkgFixedBy(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
