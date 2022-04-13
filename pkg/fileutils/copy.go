package fileutils

import (
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

// CopyNoOverwrite copies source file to destination file. If destination file exists, copying is skipped.
func CopyNoOverwrite(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return errors.Wrapf(err, "copying %q to %q. Failed to open source file", src, dst)
	}
	defer func() {
		err := in.Close()
		if err != nil {
			log.Errorf("Failed to close the file %q: %v", src, err)
		}
	}()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "copying %q to %q. Failed to open destination file", src, dst)
	}
	defer func() {
		err := out.Close()
		if err != nil {
			log.Errorf("Failed to close the file %q: %v", dst, err)
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return errors.Wrapf(err, "copying source %q to destination %q", src, dst)
	}

	return nil
}

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
