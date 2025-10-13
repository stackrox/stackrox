package npg

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// WriteFile ensures that the content is written to the path. Directories on the path are created if required.
func WriteFile(outputPath string, content string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "error creating directory for file %q", outputPath)
	}
	return errors.Wrap(os.WriteFile(outputPath, []byte(content), os.FileMode(0644)), "error writing file")
}
