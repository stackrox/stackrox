package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/facebookincubator/nvdtools/cvefeed/nvd/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestData(t *testing.T) []*schema.NVDCVEFeedJSON10DefCVEItem {
	// Fixture is generated in the following way:
	// 1. took one record from CVE list downloaded from NVD feed: https://nvd.nist.gov/vuln/data-feeds#JSON_FEED
	//    -> save single JSON record in single-cve.json
	// 2. generate all paths: https://www.convertjson.com/json-path-list.htm
	//    -> save result in all-paths.txt
	// 3. generate versions without each path with the following command:
	// cat all-paths.txt | awk '{print "jq '"'"'del(" $0 ")'"'"' -c single-cve.json >> test-fixture-cve-list.json"}' | xargs -0 bash -c
	// 4. add null record and array wrapper manually
	b, err := os.ReadFile("test-fixture-cve-list.json")
	require.NoError(t, err)

	var cveEntries []*schema.NVDCVEFeedJSON10DefCVEItem
	err = json.Unmarshal(b, &cveEntries)
	require.NoError(t, err)

	return cveEntries
}

func TestNvdCVEToEmbeddedCVE(t *testing.T) {
	i := 0
	defer func() {
		assert.Nil(t, recover(), fmt.Sprintf("NvdCVEToEmbeddedCVE panicked for entry %d in CVE list", i))
	}()

	testData := getTestData(t)
	for _, testRecord := range testData {
		_, _ = NvdCVEToEmbeddedCVE(testRecord, Istio)

		i++
	}

	// Ensure that all records are processed.
	assert.Equal(t, 73, i)
}
