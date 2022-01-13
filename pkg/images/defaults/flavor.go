package defaults

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version"
)

type imageFlavorDescriptor struct {
	// ImageFlavorName is a value for both ROX_IMAGE_FLAVOR and for --image-defaults argument in roxctl.
	ImageFlavorName string
	// IsAllowedInReleaseBuild sets if given image flavor can (true) or shall not (false) be available when
	// buildinfo.ReleaseBuild is true.
	IsAllowedInReleaseBuild bool
	// ConstructorFunc is a function that creates and populates the ImageFlavor struct according to selected
	// ImageFlavorName.
	ConstructorFunc func() ImageFlavor
}

var (
	log = logging.LoggerForModule()

	// allImageFlavors describes all available image flavors.
	allImageFlavors = []imageFlavorDescriptor{
		{
			ImageFlavorName:         ImageFlavorNameDevelopmentBuild,
			IsAllowedInReleaseBuild: false,
			ConstructorFunc:         DevelopmentBuildImageFlavor,
		},
		{
			ImageFlavorName:         ImageFlavorNameStackRoxIORelease,
			IsAllowedInReleaseBuild: true,
			ConstructorFunc:         StackRoxIOReleaseImageFlavor,
		},
	}

	// imageFlavorMap contains allImageFlavors keyed by ImageFlavorName.
	imageFlavorMap = func() map[string]imageFlavorDescriptor {
		result := make(map[string]imageFlavorDescriptor, len(allImageFlavors))
		for _, f := range allImageFlavors {
			result[f.ImageFlavorName] = f
		}
		return result
	}()
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
// TODO(RS-380): Remove this function
func GetImageFlavorByBuildType() ImageFlavor {
	if buildinfo.ReleaseBuild {
		return StackRoxIOReleaseImageFlavor()
	}
	return DevelopmentBuildImageFlavor()
}

// GetAllowedImageFlavorNames returns a string slice with all accepted image flavor names for the given
// release/development state.
func GetAllowedImageFlavorNames(isReleaseBuild bool) []string {
	result := make([]string, 0, len(allImageFlavors))
	for _, f := range allImageFlavors {
		if f.IsAllowedInReleaseBuild || !isReleaseBuild {
			result = append(result, f.ImageFlavorName)
		}
	}
	return result
}

// CheckImageFlavorName returns error if image flavor name is unknown or not allowed for the selected type of build
// (release==true, development==false), returns nil otherwise.
func CheckImageFlavorName(imageFlavorName string, isReleaseBuild bool) error {
	valids := GetAllowedImageFlavorNames(isReleaseBuild)
	contains := false
	for _, v := range valids {
		if v == imageFlavorName {
			contains = true
			break
		}
	}
	if !contains {
		return errors.Errorf("unexpected value '%s', allowed values are %v", imageFlavorName, valids)
	}
	return nil
}

// GetImageFlavorByName returns ImageFlavor struct created for the provided flavorName if the name is valid, otherwise
// it returns an error.
func GetImageFlavorByName(flavorName string, isReleaseBuild bool) (ImageFlavor, error) {
	if err := CheckImageFlavorName(flavorName, isReleaseBuild); err != nil {
		return ImageFlavor{}, err
	}
	f := imageFlavorMap[flavorName]
	return f.ConstructorFunc(), nil
}

// GetImageFlavorFromEnv returns the flavor based on the environment variable (ROX_IMAGE_FLAVOR).
// This function should be used only where this environment variable is set.
// Providing development_build flavor on a release build binary will cause the application to panic.
// We set ROX_IMAGE_FLAVOR in main and operator container images and so the code which executes in Central and operator
// can rely on GetImageFlavorFromEnv. Any code that is executed outside these images should not use this function or at
// least you should exercise great caution and check the context if ROX_IMAGE_FLAVOR is available (or make it so). For
// example, roxctl should not rely on this function because it is a standalone cli that can be run in any environment.
// roxctl should instead rely on different ways to determine which image defaults to use. Such as asking users to
// provide a command-line argument.
func GetImageFlavorFromEnv() ImageFlavor {
	f, err := GetImageFlavorByName(strings.ToLower(strings.TrimSpace(imageFlavorEnv())), buildinfo.ReleaseBuild)
	if err != nil {
		// Panic if environment variable's value is incorrect to loudly signal improper configuration of the effectively
		// build-time constant.
		log.Panicf("Incorrect image flavor in environment variable %s: %s", imageFlavorEnvName, err)
	}
	return f
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
