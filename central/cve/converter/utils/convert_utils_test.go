package utils

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNVDCVEToEmbeddedCVEs(t *testing.T) {
	data, err := os.ReadFile("testdata/cve-list.json")
	require.NoError(t, err)
	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	require.NoError(t, json.Unmarshal(data, &cveEntries))

	embeddedCVEs, err := NVDCVEsToEmbeddedCVEs(cveEntries, Istio)
	assert.NoError(t, err)
	assert.Len(t, embeddedCVEs, len(cveEntries))
}

func TestNVDCVEToEmbeddedCVE(t *testing.T) {
	data, err := os.ReadFile("testdata/cve-list.json")
	require.NoError(t, err)
	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	require.NoError(t, json.Unmarshal(data, &cveEntries))

	for _, cveEntry := range cveEntries {
		t.Run(cveEntry.CVE.CVEDataMeta.ID, func(t *testing.T) {
			embeddedCVE, err := NVDCVEToEmbeddedCVE(cveEntry, Istio)
			assert.NoError(t, err)

			assert.Equal(t, cveEntry.CVE.CVEDataMeta.ID, embeddedCVE.Cve)
			assert.True(t, embeddedCVE.CvssV2 != nil || embeddedCVE.CvssV3 != nil)
			if embeddedCVE.CvssV3 != nil {
				assert.Equal(t, storage.EmbeddedVulnerability_V3, embeddedCVE.ScoreVersion)
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV3.CVSSV3.BaseScore), embeddedCVE.CvssV3.Score)
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV3.CVSSV3.BaseScore), embeddedCVE.Cvss)
			} else {
				assert.Equal(t, storage.EmbeddedVulnerability_V2, embeddedCVE.ScoreVersion)
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV2.CVSSV2.BaseScore), embeddedCVE.CvssV2.Score)
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV2.CVSSV2.BaseScore), embeddedCVE.Cvss)
			}
			assert.Equal(t, storage.EmbeddedVulnerability_ISTIO_VULNERABILITY, embeddedCVE.VulnerabilityType)
			assert.Equal(t, cveEntry.CVE.Description.DescriptionData[0].Value, embeddedCVE.Summary)
		})
	}
}
