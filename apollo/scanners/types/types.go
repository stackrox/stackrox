package types

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// ImageScanner is the interface that all scanners must implement
type ImageScanner interface {
	GetScan(id string) (*v1.ImageScan, error)
	Scan(id string) error // Potentially initiate scan
}
