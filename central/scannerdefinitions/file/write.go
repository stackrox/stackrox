package file

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

const tempFilePattern = "scanner-defs-download-*"

// Write writes the contents of r into the path represented by the given file.
// The file's modified time is set to the given modifiedTime.
// Write is thread-safe.
func Write(file *Metadata, r io.Reader, modifiedTime time.Time) error {
	dir := filepath.Dir(file.GetPath())

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "creating subdirectory for scanner defs")
	}
	// Write the contents of r into a temporary destination to prevent us from holding the lock
	// while reading from r. The reader may be dependent on the network, and we do not want to
	// lock while depending on something as unpredictable as the network.
	// Rename is only guaranteed to be atomic inside the same directory.
	scannerDefsFile, err := os.CreateTemp(dir, tempFilePattern)
	if err != nil {
		return errors.Wrap(err, "creating scanner defs file")
	}
	_, err = io.Copy(scannerDefsFile, r)
	if err != nil {
		return errors.Wrap(err, "copying scanner defs zip out")
	}
	err = os.Chtimes(scannerDefsFile.Name(), time.Now(), modifiedTime)
	if err != nil {
		return errors.Wrap(err, "changing modified time of scanner defs")
	}

	// Note: os.Rename does not alter the file's modified time,
	// so there is no need to call os.Chtimes here.
	err = os.Rename(scannerDefsFile.Name(), file.GetPath())
	if err != nil {
		return errors.Wrap(err, "renaming scanner defs file")
	}

	return nil
}
