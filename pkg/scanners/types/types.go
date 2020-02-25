package types

import (
	"github.com/stackrox/rox/generated/storage"
)

// ImageScanner is the interface that all scanners must implement
type ImageScanner interface {
	ScanSemaphore

	// GetScan gets the scan for the given image.
	// It is a blocking call; if the scanner has not scanned the image yet,
	// the function blocks until it does. It returns an error if it fails to do so.
	GetScan(image *storage.Image) (*storage.ImageScan, error)
	Match(image *storage.Image) bool
	Test() error
	Type() string
	Name() string
}

// AsyncImageScanner is an image scanner that can be accessed asynchronously.
type AsyncImageScanner interface {
	ImageScanner
	// GetOrTriggerScan does a non-blocking request to the scanner.
	// It gets the scan for the given image if it exists;
	// if not, implementations trigger a new one and instantly return.
	GetOrTriggerScan(image *storage.Image) (*storage.ImageScan, error)
}
