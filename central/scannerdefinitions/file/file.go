package file

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

const tempFilePattern = "scanner-defs-download-*"

// File is a wrapper around a file path
// which exposes a thread-safe Open and Write API.
type File struct {
	path string
}

// New creates a new File.
func New(path string) *File {
	return &File{
		path: path,
	}
}

// Path returns the path of the referenced file.
func (file *File) Path() string {
	return file.path
}

// Write writes the contents of r into the path represented by the given file.
// The file's modified time is set to the given modifiedTime.
// Write is thread-safe.
func (file *File) Write(r io.Reader, modifiedTime time.Time) error {
	dir := filepath.Dir(file.path)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "creating subdirectory for scanner defs")
	}

	// Write the contents of r into a temporary destination to prevent us from having to hold a lock
	// while reading from r. The reader may be dependent on the network, and we do not want to
	// lock while depending on something as unpredictable as the network.
	scannerDefsFile, err := os.CreateTemp(dir, tempFilePattern)
	if err != nil {
		return errors.Wrap(err, "creating scanner defs file")
	}
	// Close the file in case of error.
	defer utils.IgnoreError(scannerDefsFile.Close)

	_, err = io.Copy(scannerDefsFile, r)
	if err != nil {
		return errors.Wrapf(err, "writing scanner defs zip to temporary file %q", scannerDefsFile.Name())
	}

	// No longer need the file descriptor, so release it.
	// Closing here, as it is possible Close updates the mtime
	// (for example: the data is not flushed until Close is called).
	err = scannerDefsFile.Close()
	if err != nil {
		return errors.Wrap(err, "closing temp scanner defs file")
	}

	err = os.Chtimes(scannerDefsFile.Name(), time.Now(), modifiedTime)
	if err != nil {
		return errors.Wrap(err, "changing modified time of scanner defs")
	}

	// Note: os.Rename does not alter the file's modified time,
	// so there is no need to call os.Chtimes here.
	// Rename is guaranteed to be atomic inside the same directory.
	err = os.Rename(scannerDefsFile.Name(), file.path)
	if err != nil {
		return errors.Wrap(err, "renaming temporary scanner defs file to final location")
	}

	return nil
}

// Open opens the file at the given path and returns the contents and modified time.
// If the file does not exist, it is *not* an error. In this case, nil is returned.
// It is the caller's responsibility to close the returned file.
// Open is thread-safe.
func (file *File) Open() (*os.File, time.Time, error) {
	// This is thread-safe due to the semantics of rename().
	// See the manpage for more information: https://linux.die.net/man/3/rename
	f, err := os.Open(file.path)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return nil, time.Time{}, nil
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	var succeeded bool
	defer func() {
		if !succeeded {
			// Release the file descriptor, as there was an error.
			utils.IgnoreError(f.Close)
		}
	}()

	fi, err := f.Stat()
	if err != nil {
		return nil, time.Time{}, err
	}

	succeeded = true
	return f, fi.ModTime().UTC(), nil
}
