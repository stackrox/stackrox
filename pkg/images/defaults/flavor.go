package defaults

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version"
)

var (
	log = logging.LoggerForModule()
)

// ChartRepo contains information about where the Helm charts are published.
type ChartRepo struct {
	URL string
}

// ImagePullSecrets represents the image pull secret defaults.
type ImagePullSecrets struct {
	AllowNone bool
}

// ImageFlavor represents default settings for pulling images.
type ImageFlavor struct {
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

// DevelopmentBuildImageFlavor returns image values for `development_build` flavor.
// Assumption: development_build flavor is never a release.
func DevelopmentBuildImageFlavor() ImageFlavor {
	v := version.GetAllVersionsDevelopment()
	return ImageFlavor{
		MainRegistry:  "docker.io/stackrox",
		MainImageName: "main",
		MainImageTag:  v.MainVersion,

		CollectorRegistry:      "docker.io/stackrox",
		CollectorImageName:     "collector",
		CollectorImageTag:      v.CollectorVersion + "-latest",
		CollectorSlimImageName: "collector",
		CollectorSlimImageTag:  v.CollectorVersion + "-slim",

		ScannerImageName:   "scanner",
		ScannerImageTag:    v.ScannerVersion,
		ScannerDBImageName: "scanner-db",
		ScannerDBImageTag:  v.ScannerVersion,

		ChartRepo: ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: true,
		},
		Versions: v,
	}
}

// StackRoxIOReleaseImageFlavor returns image values for `stackrox_io_release` flavor.
func StackRoxIOReleaseImageFlavor() ImageFlavor {
	v := version.GetAllVersionsUnified()
	return ImageFlavor{
		MainRegistry:  "stackrox.io",
		MainImageName: "main",
		MainImageTag:  v.MainVersion,

		CollectorRegistry:      "collector.stackrox.io",
		CollectorImageName:     "collector",
		CollectorImageTag:      v.CollectorVersion,
		CollectorSlimImageName: "collector-slim",
		CollectorSlimImageTag:  v.CollectorVersion,

		ScannerImageName:   "scanner",
		ScannerImageTag:    v.ScannerVersion,
		ScannerDBImageName: "scanner-db",
		ScannerDBImageTag:  v.ScannerVersion,

		ChartRepo: ChartRepo{
			URL: "https://charts.stackrox.io",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: false,
		},
		Versions: v,
	}
}

// GetImageFlavorByBuildType returns the flavor based on build type (development or release). Release builds use StackroxIO
// flavor and development builds use development flavor.
func GetImageFlavorByBuildType() ImageFlavor {
	if buildinfo.ReleaseBuild {
		return StackRoxIOReleaseImageFlavor()
	}
	return DevelopmentBuildImageFlavor()
}

// GetImageFlavorFromEnv returns the flavor based on the environment variable (ROX_IMAGE_FLAVOR). This should be used
// only where this environment variable is set, otherwise it will be defaulted to stackrox_io_release. Providing
// development_build flavor on a release build binary will cause the application to panic.
// We set ROX_IMAGE_FLAVOR in main and operator container images and so the code which executes in Central and operator
// can rely on GetImageFlavorFromEnv. Any code that is executed outside these images should not use this function or at
// least you should exercise great caution and check the context if ROX_IMAGE_FLAVOR is available (or make it so). For
// example, roxctl should not rely on this function because it is a standalone cli that can be run in any environment.
// roxctl should instead rely on different ways to determine which image defaults to use. Such as asking users to
// provide a command-line argument.
func GetImageFlavorFromEnv() ImageFlavor {
	envValue := ImageFlavorFromEnv()
	if buildinfo.ReleaseBuild && envValue == imageFlavorDevelopment {
		// Release product build using development image repositories is likely a misconfiguration. We don't want to
		// accidentally go out with development images into release.
		log.Panicf("Cannot use %s flavor in build release", imageFlavorDevelopment)
	}

	imageFlavorMap := map[string]func() ImageFlavor{
		imageFlavorDevelopment: DevelopmentBuildImageFlavor,
		imageFlavorStackroxIO:  StackRoxIOReleaseImageFlavor,
	}

	if fn, ok := imageFlavorMap[ImageFlavorFromEnv()]; ok {
		return fn()
	}

	log.Warnf("Environment variable %s has invalid value %s. Using default image flavor %s",
		imageFlavorEnvName,
		envValue,
		"stackroxio_release")
	return StackRoxIOReleaseImageFlavor()
}

// IsImageDefaultMain checks if provided image matches main image defined in flavor.
func (f *ImageFlavor) IsImageDefaultMain(img *storage.ImageName) bool {
	overrideImageNoTag := fmt.Sprintf("%s/%s", img.Registry, img.Remote)
	return f.MainImageNoTag() == overrideImageNoTag
}

// ScannerImage is the container image reference (full name) for the scanner image.
func (f *ImageFlavor) ScannerImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.ScannerImageName, f.ScannerImageTag)
}

// ScannerDBImage is the container image reference (full name) for the scanner-db image.
func (f *ImageFlavor) ScannerDBImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.ScannerDBImageName, f.ScannerDBImageTag)
}

// MainImage is the container image reference (full name) for the "main" image.
func (f *ImageFlavor) MainImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.MainImageName, f.MainImageTag)
}

// MainImageNoTag is the container image repository (image name including registry, excluding tag) for the "main" image.
func (f *ImageFlavor) MainImageNoTag() string {
	return fmt.Sprintf("%s/%s", f.MainRegistry, f.MainImageName)
}

// CollectorFullImage is the container image reference (full name) for the "collector" image
func (f *ImageFlavor) CollectorFullImage() string {
	return fmt.Sprintf("%s/%s:%s", f.CollectorRegistry, f.CollectorImageName, f.CollectorImageTag)
}

// CollectorSlimImage is the container image reference (full name) for the "collector slim" image
func (f *ImageFlavor) CollectorSlimImage() string {
	return fmt.Sprintf("%s/%s:%s", f.CollectorRegistry, f.CollectorSlimImageName, f.CollectorSlimImageTag)
}

// CollectorFullImageNoTag is the container image repository (image name including registry, excluding tag) for the  "collector" image.
func (f *ImageFlavor) CollectorFullImageNoTag() string {
	return fmt.Sprintf("%s/%s", f.CollectorRegistry, f.CollectorImageName)
}

// CollectorImageNoTag is the container image repository (image name including registry, excluding tag) for the "collector" image.
func (f *ImageFlavor) CollectorImageNoTag() string {
	return fmt.Sprintf("%s/%s", f.CollectorRegistry, f.CollectorImageName)
}
