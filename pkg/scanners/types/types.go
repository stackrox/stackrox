package types

import "github.com/stackrox/rox/generated/api/v1"

// ImageScanner is the interface that all scanners must implement
type ImageScanner interface {
	GetLastScan(image *v1.Image) (*v1.ImageScan, error)
	Match(image *v1.Image) bool
	Test() error
	Global() bool
}
