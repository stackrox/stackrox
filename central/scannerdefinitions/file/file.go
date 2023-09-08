package file

import (
	"bytes"
	"io"
	"io/fs"
	"log"
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

func (file *File) Write(r io.Reader, modifiedTime time.Time) error {
	err := file.writeInternal(r)
	if err != nil {
		return err
	}

	// Set the file's modified time after the write is completed
	err = os.Chtimes(file.path, time.Now(), modifiedTime)
	if err != nil {
		return errors.Wrap(err, "changing modified time of scanner defs")
	}

	return nil
}

func (file *File) WriteContent(r io.Reader) error {
	return file.writeInternal(r)
}

func (file *File) writeInternal(r io.Reader) error {
	// Check if r is empty by trying to read a single byte.
	buf := make([]byte, 1)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		log.Println(err.Error())
		return errors.Wrap(err, "checking if reader is empty")
	}

	// If no bytes are read, then the reader is empty.
	if n == 0 {
		log.Println(errors.New("provided reader is empty").Error())
		return errors.New("provided reader is empty")
	}

	// Combine the read byte with the original reader.
	r = io.MultiReader(bytes.NewReader(buf[:n]), r)

	dir := filepath.Dir(file.path)

	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.Wrap(err, "creating subdirectory for scanner defs")
	}

	scannerDefsFile, err := os.CreateTemp(dir, tempFilePattern)
	if err != nil {
		return errors.Wrap(err, "creating scanner defs file")
	}
	defer utils.IgnoreError(scannerDefsFile.Close)

	_, err = io.Copy(scannerDefsFile, r)
	if err != nil {
		return errors.Wrapf(err, "writing scanner defs zip to temporary file %q", scannerDefsFile.Name())
	}

	err = scannerDefsFile.Close()
	if err != nil {
		return errors.Wrap(err, "closing temp scanner defs file")
	}

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
