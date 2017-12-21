package types

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

// ImageScanner is the interface that all scanners must implement
type ImageScanner interface {
	ProtoScanner() *v1.Scanner

	GetLastScan(image *v1.Image) (*v1.ImageScan, error)
	Match(image *v1.Image) bool
	Test() error
}
