package types

import (
	"github.com/stackrox/rox/generated/storage"
)

// ImageScanner is the interface that all scanners must implement
type ImageScanner interface {
	GetLastScan(image *storage.Image) (*storage.ImageScan, error)
	Match(image *storage.Image) bool
	Test() error
	Global() bool
	Type() string
}
