package mapkubeapis

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/stackrox/rox/pkg/utils"
)

const tempFilePattern = "mapkubeapis-map-*.yaml"

var (
	//go:embed Map.yaml
	embeddedMapFile []byte
)

// CreateTempMapFile creates a temporary map file from the embedded FS. Returns the path to a created temporary file.
func CreateTempMapFile() (string, error) {
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
