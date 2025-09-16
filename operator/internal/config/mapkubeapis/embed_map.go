package mapkubeapis

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const tempFilePattern = "mapkubeapis-map-*.yaml"

var (
	//go:embed Map.yaml
	embeddedMapFile []byte

	tempMapFile string

	once sync.Once
)

// GetMapFilePath returns the path to a newly created or existing temporary file.
func GetMapFilePath() string {
	once.Do(func() {
		tempFile, err := createTempMapFile()
		utils.Should(err)
		tempMapFile = tempFile
	})
	return tempMapFile
}

func createTempMapFile() (string, error) {
	tempFile, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		return "", fmt.Errorf("creating mapkubeapis map file: %w", err)
	}
	defer utils.IgnoreError(tempFile.Close)

	if err := os.WriteFile(tempFile.Name(), embeddedMapFile, 0644); err != nil {
		return "", fmt.Errorf("writing embeded mapkubeapis map file: %q", tempFile.Name())
	}

	return tempFile.Name(), nil
}
