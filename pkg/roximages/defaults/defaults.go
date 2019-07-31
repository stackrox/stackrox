package defaults

import (
	"fmt"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/version"
)

// ScannerV2DBImage is the Docker image name for the DB we use with scanner v2.
func ScannerV2DBImage() string {
	return fmt.Sprintf("%s/scanner-v2-db:%s", getRegistry(), version.GetScannerV2Version())
}

// ScannerImage is the Docker image name for the scanner image. Image
// repo changes depending on whether or not this is a release build.
func ScannerImage() string {
	return fmt.Sprintf("%s/scanner:%s", getRegistry(), version.GetScannerVersion())
}

// ScannerV2Image is the Docker image name for the scanner v2 image. Image
// repo changes depending on whether or not this is a release build.
func ScannerV2Image() string {
	return fmt.Sprintf("%s/scanner-v2:%s", getRegistry(), version.GetScannerV2Version())
}

// MainImage is the Docker image name for the "main" image. Image repo
// changes depending on whether or not this is a release build.
func MainImage() string {
	return fmt.Sprintf("%s:%s", MainImageRepo(), version.GetMainVersion())
}

// MainImageRepo is the Docker image repo for the "main" image. It
// changes depending on whether or not this is a release build.
func MainImageRepo() string {
	return getRegistry() + "/main"
}

func getRegistry() string {
	if buildinfo.ReleaseBuild {
		return "stackrox.io"
	}
	return "docker.io/stackrox"
}
