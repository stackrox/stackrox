package scancomponent

import (
	"bufio"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"google.golang.org/protobuf/encoding/protojson"
)

var update = flag.Bool("update", false, "update golden files")

//go:embed testdata_ComponentIDV2.json
var testdata_ComponentIDV2 []byte

// TestConsistentComponentIDV2 is using a golden file, meaning a file hashed once on creation of this test
// to verify ComponentIDV2s generated for the test image don't change over time.
// If a change of the ComponentIDV2s is expected run this test with the -update flag to update the golden file
func TestConsistentComponentIDV2(t *testing.T) {
	var image storage.Image
	if err := protojson.Unmarshal(testdata_ComponentIDV2, &image); err != nil {
		t.Fatalf("Failed to unmarshal image data: %v", err)
	}

	imageID := image.GetId()
	components := image.GetScan().GetComponents()

	if len(components) == 0 {
		t.Fatal("No components found in testdata")
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
		err := writeGoldenFile(ids)
		if err != nil {
			t.Fatal(err)
		}
		return
	}

	goldenIDs, err := readGoldenFile()
	if err != nil {
		t.Fatalf("Failed to read golden file: %v\nRun with -update flag to create/update the golden file", err)
	}

	if len(ids) != len(goldenIDs) {
		t.Fatalf("Component count mismatch: got %d, want %d", len(ids), len(goldenIDs))
	}

	var diffs []string
	for i := range ids {
		if ids[i] != goldenIDs[i] {
			diffs = append(diffs, fmt.Sprintf("Line %d:\n  got:  %s\n  want: %s", i+1, ids[i], goldenIDs[i]))
		}
	}

	if len(diffs) > 0 {
		t.Fatalf("Component IDs don't match golden file:\n%s", strings.Join(diffs, "\n"))
	}
}

func writeGoldenFile(ids []string) error {
	filePath := "testdata_ComponentIDV2Golden.txt"
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to write golden file at path: %s, err: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for _, id := range ids {
		_, err := writer.WriteString(id + "\n")
		if err != nil {
			return fmt.Errorf("error writing to golden file: %s, err: %w", filePath, err)
		}
	}

	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("failed to flush content to file: %s, err: %w", filePath, err)
	}

	return nil
}

func readGoldenFile() ([]string, error) {
	filePath := "testdata_ComponentIDV2Golden.txt"
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open golden file at path: %s, err: %w", filePath, err)
	}
	defer file.Close()

	var ids []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			ids = append(ids, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading golden file: %s, err: %w", filePath, err)
	}

	return ids, nil
}

func BenchmarkComponentIDV2Test(b *testing.B) {
	var image storage.Image

	if err := protojson.Unmarshal(testdata_ComponentIDV2, &image); err != nil {
		b.Fatalf("Failed to unmarshal image data: %v", err)
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
