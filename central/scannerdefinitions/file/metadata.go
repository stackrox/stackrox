package file

import (
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

// Metadata represents file metadata required to read/write a file.
// It implements the sync.RWMutex interface so users may lock the file.
type Metadata struct {
	path             string
	lastModifiedTime time.Time

	sync.RWMutex
}

// NewMetadata creates a new FileMetadata given a path with the given
// modified time.
func NewMetadata(path string, lastModifiedTime *time.Time) *Metadata {
	var t time.Time
	if lastModifiedTime != nil {
		t = *lastModifiedTime
	}
	return &Metadata{
		path:             path,
		lastModifiedTime: t,
	}
}

// GetPath returns the path for the file.
func (m *Metadata) GetPath() string {
	return m.path
}

// GetLastModifiedTime returns the last modified time of the file, in UTC.
func (m *Metadata) GetLastModifiedTime() time.Time {
	return m.lastModifiedTime.UTC()
}

// SetLastModifiedTime sets the last modified time of the file.
func (m *Metadata) SetLastModifiedTime(lastModifiedTime time.Time) {
	m.lastModifiedTime = lastModifiedTime
}
