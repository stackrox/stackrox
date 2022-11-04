package cache

import (
	"archive/zip"
	"time"
)

// Cache is the interface for common cache operations.
type Cache interface {
	Dir() string
	LoadFromDirectory(definitionsDir string) error
	LoadFromZip(zipR *zip.ReadCloser, definitionsDir string) error
	GetLastUpdate() time.Time
	SetLastUpdate(t time.Time)
}
