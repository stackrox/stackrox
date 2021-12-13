package images

import (
	"fmt"
)

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

// MainImageUntagged is the Docker image repo for the "main" image. It
// changes depending on whether this is a release build.
func (flavor *Flavor) MainImageUntagged() string {
	return fmt.Sprintf("%s/%s", flavor.MainRegistry, flavor.MainImageName)
}
