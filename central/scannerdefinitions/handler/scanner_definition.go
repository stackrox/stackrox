package handler

import (
	"os"
	"time"
)

// vulDefFile unifies online and offline reader for scanner definitions
type vulDefFile struct {
	*os.File
	modTime time.Time
	closer  func() error
}

func (f *vulDefFile) Close() error {
	if f.closer != nil {
		return f.closer()
	}
	return f.File.Close()
}
