package npguardutils

import (
	"errors"
	"os"
	"path/filepath"

	e "github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	// ErrNPGErrorsIndicator errors indicator message
	ErrNPGErrorsIndicator = errors.New("there were errors during execution")
	// ErrNPGWarningsIndicator warnings indicator message
	ErrNPGWarningsIndicator = errors.New("there were warnings during execution")
)

// WriteFile writes output file for the relevant NP-Guard command
func WriteFile(filename string, destDir string, content string) error {
	outputPath := filepath.Join(destDir, filename)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return e.Wrapf(err, "error creating directory for file %q", filename)
	}
	return e.Wrap(os.WriteFile(outputPath, []byte(content), os.FileMode(0644)), "error writing file")
}

// SetupPath checks if possible to set the output path
func SetupPath(path string, removedOutputPathFlag bool) error {
	if _, err := os.Stat(path); err == nil && !removedOutputPathFlag {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return e.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}
