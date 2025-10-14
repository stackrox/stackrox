package cve

import (
	_ "embed"
	"flag"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/testutils/hashdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCVETypesAreAccountedFor(t *testing.T) {
	// + 1 for unknown type
	assert.Equal(t, len(storage.CVE_CVEType_name), len(clusterCVETypes)+len(componentCVETypes)+1)
}

var update = flag.Bool("update", false, "update golden files")

// TestConsitentIDV2 is using a golden file, meaning a file containing IDs generated at creation of this test
// to verify IDV2s generated for the test image don't change over time.
// If a change of the IDV2s is expected run this test with the -update flag to update the golden file
func TestConsistentIDV2(t *testing.T) {
	goldenFilePath := "testdata_IDV2Golden.txt"
	image, err := hashdata.GetImage()
	if err != nil {
		t.Fatalf("failed to get image data: %v", err)
	}

	imageID := image.GetId()
	components := image.GetScan().GetComponents()

	vulnIDs := []string{}
	if len(components) == 0 {
		t.Fatal("no components found in testdata")
	}

	for _, c := range components {
		cID, err := scancomponent.ComponentIDV2(c, imageID)
		require.NoError(t, err, "failed to hash component")

		for _, vuln := range c.GetVulns() {
			vID, err := IDV2(vuln, cID)
			require.NoError(t, err, "failed to hash vulnerability")
			vulnIDs = append(vulnIDs, vID)
		}
	}

	if *update {
		err := hashdata.WriteLinesToFile(goldenFilePath, vulnIDs)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	goldenIDs, err := hashdata.ReadLinesFromFile(goldenFilePath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v\nRun with -update flag to create/update the golden file", err)
	}

	require.Equal(t, vulnIDs, goldenIDs)
}
