package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestData(t *testing.T) []*schema.NVDCVEFeedJSON10DefCVEItem {
	b, err := os.ReadFile("testdata/istio/cve-list.json")
	require.NoError(t, err)

	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	err = json.Unmarshal(b, &cveEntries)
	require.NoError(t, err)

	return cveEntries
}

func getPanicTestData(t *testing.T) []*schema.NVDCVEFeedJSON10DefCVEItem {
	b, err := os.ReadFile("testdata/istio/cve-list-panic.json")
	require.NoError(t, err)

	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	err = json.Unmarshal(b, &cveEntries)
	require.NoError(t, err)

	return cveEntries
}

func TestNVDCVEToEmbeddedCVEs_Istio(t *testing.T) {
	cveEntries := getTestData(t)

	embeddedCVEs, err := NVDCVEsToEmbeddedCVEs(cveEntries, Istio)
	assert.NoError(t, err)
	assert.Len(t, embeddedCVEs, len(cveEntries))
}

func TestNVDCVEToEmbeddedCVE_Istio(t *testing.T) {
	cveEntries := getTestData(t)

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

func TestNVDCVEToEmbeddedCVE_Panic(t *testing.T) {
	var i int
	defer func() {
		assert.Nil(t, recover(), fmt.Sprintf("NVDCVEToEmbeddedCVE panicked for entry %d in CVE list", i))
	}()

	testData := getPanicTestData(t)
	for _, testRecord := range testData {
		_, _ = NVDCVEToEmbeddedCVE(testRecord, Istio)

		i++
	}

	// Ensure that all records are processed.
	assert.Equal(t, len(testData), i)
}
