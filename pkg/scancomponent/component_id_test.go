package scancomponent

import (
	_ "embed"
	"flag"
	"testing"

	"github.com/stackrox/rox/pkg/testutils/hashdata"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update golden files")

// TestConsitentComponentIDV2 is using a golden file, meaning a file containing IDs generated at creation of this test
// to verify ComponentIDV2s generated for the test image don't change over time.
// If a change of the ComponentIDV2s is expected run this test with the -update flag to update the golden file
func TestConsistentComponentIDV2(t *testing.T) {
	goldenFilePath := "testdata_ComponentIDV2Golden.txt"
	image, err := hashdata.GetImage()
	if err != nil {
		t.Fatalf("failed to get image data: %v", err)
	}

	imageID := image.GetId()
	components := image.GetScan().GetComponents()

	if len(components) == 0 {
		t.Fatal("no components found in testdata")
	}

	ids := make([]string, len(components))

	for i, component := range components {
		id, err := ComponentIDV2(component, imageID)
		if err != nil {
			t.Fatalf("ComponentIDV2 failed: %v", err)
		}

		ids[i] = id
	}

	if *update {
		err := hashdata.WriteLinesToFile(goldenFilePath, ids)
		if err != nil {
			t.Fatalf("failed to write golden file at %s: %v", goldenFilePath, err)
		}
		return
	}

	goldenIDs, err := hashdata.ReadLinesFromFile(goldenFilePath)
	if err != nil {
		t.Fatalf("failed to read golden file: %v\nRun with -update flag to create/update the golden file", err)
	}

	require.Equal(t, ids, goldenIDs)
}

func BenchmarkComponentIDV2Test(b *testing.B) {
	image, err := hashdata.GetImage()
	if err != nil {
		b.Fatalf("failed to get image data: %v", err)
	}

	imageID := image.GetId()
	components := image.GetScan().GetComponents()

	if len(components) == 0 {
		b.Fatal("No components found in image")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, component := range components {
			_, err := ComponentIDV2(component, imageID)
			if err != nil {
				b.Fatalf("ComponentIDV2 failed: %v", err)
			}
		}
	}
}
