package types

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// ImageScanner is the interface that all scanners must implement
type ImageScanner interface {
	Config() map[string]string
	Endpoint() string

	GetLastScan(image *v1.Image) (*v1.ImageScan, error)
	GetScans(image *v1.Image) ([]*v1.ImageScan, error)
	Scan(image *v1.Image) error // Potentially initiate scan
	Test() error
}
