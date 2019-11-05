package fetcher

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stackrox/k8s-istio-cve-pusher/nvd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	correctCVEFile  = "testdata/correct_cves.json"
	cveChecksumFile = "testdata/cve_checksum"
)

func TestUnmarshalCorrectCVEs(t *testing.T) {
	dat, err := ioutil.ReadFile(correctCVEFile)
	require.Nil(t, err)
	var cveEntries []nvd.CVEEntry
	err = json.Unmarshal(dat, &cveEntries)
	assert.Nil(t, err)
	assert.Len(t, cveEntries, 2)
}

func TestReadChecksum(t *testing.T) {
	data, err := ioutil.ReadFile(cveChecksumFile)
	require.Nil(t, err)
	assert.Equal(t, string(data), "e76a63173f5b1e8bdcc9811faf4a4643266cdcb1e179229e30ffcb0e5d8dbe0c")
}
