package file

import (
	"os"
)

// Read returns the *os.File and os.FileInfo structure describing the file.
// Read is not thread-safe.
func Read(file *Metadata) (*os.File, os.FileInfo, error) {
	f, err := os.Open(file.GetPath())
	if err != nil {
		return nil, nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}

	return f, fi, nil
}
