package images

import (
	"fmt"
)

// Set of utility methods used across the applications and roxctl to determine image values based on flavor.

// ScannerImage is the Docker image name for the scanner image. Image
// repo changes depending on whether this is a release build.
func (flavor *Flavor) ScannerImage() string {
	return fmt.Sprintf("%s/%s:%s", flavor.MainRegistry, flavor.ScannerImageName, flavor.ScannerImageTag)
}

// ScannerDBImage is the Docker image name for the scanner db image
func (flavor *Flavor) ScannerDBImage() string {
	return fmt.Sprintf("%s/%s:%s", flavor.MainRegistry, flavor.ScannerDBImageName, flavor.ScannerDBImageTag)
}

// MainImage is the Docker image name for the "main" image. Image repo
// changes depending on whether this is a release build.
func (flavor *Flavor) MainImage() string {
	return fmt.Sprintf("%s/%s:%s", flavor.MainRegistry, flavor.MainImageName, flavor.MainImageTag)
}

// MainImageRepo is the Docker image repo for the "main" image. It
// changes depending on whether this is a release build.
func (flavor *Flavor) MainImageRepo() string {
	return fmt.Sprintf("%s/%s", flavor.MainRegistry, flavor.MainImageName)
}

// MainImageRegistry is the Docker image registry for the "main" image.
func (flavor *Flavor) MainImageRegistry() string {
	return flavor.MainRegistry
}

// CollectorImageRegistry is the Docker image registry for the "collector" image.
func (flavor *Flavor) CollectorImageRegistry() string {
	return flavor.CollectorRegistry
}
