package fileutils

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// CopySrcToFile copies the content from supplied reader to destination file.
func CopySrcToFile(file string, src io.Reader) error {
	f, err := os.Create(file)
	if err != nil {
		return errors.Wrap(err, "creating file")
	}
	defer utils.IgnoreError(f.Close)

	_, err = io.Copy(f, src)
	if err != nil {
		return errors.Wrap(err, "writing to file")
	}
	return nil
}
