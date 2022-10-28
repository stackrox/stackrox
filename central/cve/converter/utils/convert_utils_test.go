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

func TestNVDCVEToEmbeddedCVE_Istio(t *testing.T) {
	data, err := os.ReadFile("testdata/istio/cve-list.json")
	require.NoError(t, err)
	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	require.NoError(t, json.Unmarshal(data, &cveEntries))

	for _, cveEntry := range cveEntries {
		cveEntry := cveEntry
		require.NotNil(t, cveEntry)
		require.NotNil(t, cveEntry.Impact)
		require.True(t, cveEntry.Impact.BaseMetricV2 != nil || cveEntry.Impact.BaseMetricV3 != nil)

		t.Run(cveEntry.CVE.CVEDataMeta.ID, func(t *testing.T) {
			embeddedCVE, err := NVDCVEToEmbeddedCVE(cveEntry, Istio)
			assert.NoError(t, err)

			assert.Equal(t, cveEntry.CVE.CVEDataMeta.ID, embeddedCVE.GetCve())
			assert.True(t, embeddedCVE.GetCvssV2() != nil || embeddedCVE.GetCvssV3() != nil)
			if cveEntry.Impact.BaseMetricV3 != nil {
				assert.Equal(t, storage.EmbeddedVulnerability_V3, embeddedCVE.GetScoreVersion())
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV3.CVSSV3.BaseScore), embeddedCVE.GetCvssV3().GetScore())
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV3.CVSSV3.BaseScore), embeddedCVE.GetCvss())
				assert.Equal(t, cveEntry.Impact.BaseMetricV3.CVSSV3.VectorString, embeddedCVE.GetCvssV3().GetVector())
			} else {
				assert.Equal(t, storage.EmbeddedVulnerability_V2, embeddedCVE.GetScoreVersion())
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV2.CVSSV2.BaseScore), embeddedCVE.GetCvssV2().GetScore())
				assert.Equal(t, float32(cveEntry.Impact.BaseMetricV2.CVSSV2.BaseScore), embeddedCVE.GetCvss())
				assert.Equal(t, cveEntry.Impact.BaseMetricV2.CVSSV2.VectorString, embeddedCVE.GetCvssV2().GetVector())
			}
			assert.Equal(t, storage.EmbeddedVulnerability_ISTIO_VULNERABILITY, embeddedCVE.VulnerabilityType)
			assert.Equal(t, cveEntry.CVE.Description.DescriptionData[0].Value, embeddedCVE.Summary)
		})
	}
}
