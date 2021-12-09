package images

import (
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version"
)

// ChartRepo contains information about where the Helm charts are published.
type ChartRepo struct {
	URL string
}

// ImagePullSecrets represents the image pull secret defaults.
type ImagePullSecrets struct {
	AllowNone bool
}

// Flavor represents default settings for pulling images.
type Flavor struct {
	// MainRegistry is a registry for all images except of collector.
	MainRegistry  string
	MainImageName string
	MainImageTag  string

	// CollectorRegistry may be different from MainRegistry in case of stackrox.io.
	CollectorRegistry      string
	CollectorImageName     string
	CollectorImageTag      string
	CollectorSlimImageName string
	CollectorSlimImageTag  string

	ScannerImageName   string
	ScannerImageTag    string
	ScannerDBImageName string
	ScannerDBImageTag  string

	ChartRepo        ChartRepo
	ImagePullSecrets ImagePullSecrets
	Versions         version.Versions
}

var (
	log = logging.LoggerForModule()
)

// DevelopmentBuildImageFlavor returns image values for `development_build` flavor.
// Assumption: development_build flavor is never a release.
func DevelopmentBuildImageFlavor() Flavor {
	return Flavor{
		MainRegistry:  "docker.io/stackrox",
		MainImageName: "main",
		MainImageTag:  version.GetMainVersion(),

		CollectorRegistry:      "docker.io/stackrox",
		CollectorImageName:     "collector",
		CollectorImageTag:      version.GetCollectorVersion() + "-latest",
		CollectorSlimImageName: "collector",
		CollectorSlimImageTag:  version.GetCollectorVersion() + "-slim",

		ScannerImageName:   "scanner",
		ScannerImageTag:    version.GetScannerVersion(),
		ScannerDBImageName: "scanner-db",
		ScannerDBImageTag:  version.GetScannerVersion(),

		ChartRepo: ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: true,
		},
		Versions: version.GetAllVersions(),
	}
}

// StackRoxIOReleaseImageFlavor returns image values for `stackrox_io_release` flavor.
func StackRoxIOReleaseImageFlavor() Flavor {
	return Flavor{
		MainRegistry:  "stackrox.io",
		MainImageName: "main",
		MainImageTag:  version.GetMainVersion(),

		CollectorRegistry:      "collector.stackrox.io",
		CollectorImageName:     "collector",
		CollectorImageTag:      version.GetCollectorVersion(),
		CollectorSlimImageName: "collector-slim",
		CollectorSlimImageTag:  version.GetCollectorVersion(),

		ScannerImageName:   "scanner",
		ScannerImageTag:    version.GetScannerVersion(),
		ScannerDBImageName: "scanner-db",
		ScannerDBImageTag:  version.GetScannerVersion(),

		ChartRepo: ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: false,
		},
		Versions: version.GetAllVersions(),
	}
}

// GetFlavorByBuildType returns the flavor based on build type (development or release). Release builds use StackroxIO
// flavor and development builds use development flavor.
func GetFlavorByBuildType() Flavor {
	if buildinfo.ReleaseBuild {
		return StackRoxIOReleaseImageFlavor()
	}
	return DevelopmentBuildImageFlavor()
}
