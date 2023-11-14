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
	// imageFlavorName is a value for both ROX_IMAGE_FLAVOR and for --image-defaults argument in roxctl.
	imageFlavorName string
	// isVisibleInReleaseBuild sets if the given image flavor shall (true) or shall not (false) be displayed to the user
	// in cli help when buildinfo.ReleaseBuild is true. Note that the flavor can still be used even though it is not
	// displayed.
	isVisibleInReleaseBuild bool
	// constructorFunc is a function that creates and populates the ImageFlavor struct according to selected
	// ImageFlavorName.
	constructorFunc func() ImageFlavor
}

var (
	log = logging.LoggerForModule()

	// allImageFlavors describes all available image flavors.
	allImageFlavors = []imageFlavorDescriptor{
		{
			imageFlavorName:         ImageFlavorNameDevelopmentBuild,
			isVisibleInReleaseBuild: false,
			constructorFunc:         DevelopmentBuildImageFlavor,
		},
		{
			// TODO(ROX-11642): This was just hidden in release builds but should go away completely.
			imageFlavorName:         ImageFlavorNameStackRoxIORelease,
			isVisibleInReleaseBuild: false,
			constructorFunc:         StackRoxIOReleaseImageFlavor,
		},
		{
			imageFlavorName:         ImageFlavorNameRHACSRelease,
			isVisibleInReleaseBuild: true,
			constructorFunc:         RHACSReleaseImageFlavor,
		},
		{
			imageFlavorName:         ImageFlavorNameOpenSource,
			isVisibleInReleaseBuild: true,
			constructorFunc:         OpenSourceImageFlavor,
		},
	}

	// imageFlavorMap contains allImageFlavors keyed by ImageFlavorName.
	imageFlavorMap = func() map[string]imageFlavorDescriptor {
		result := make(map[string]imageFlavorDescriptor, len(allImageFlavors))
		for _, f := range allImageFlavors {
			result[f.imageFlavorName] = f
		}
		return result
	}()
)

// ChartRepo contains information about where the Helm charts are published.
type ChartRepo struct {
	URL     string
	IconURL string
}

// ImagePullSecrets represents the image pull secret defaults.
type ImagePullSecrets struct {
	AllowNone bool
}

// ImageFlavor represents default settings for pulling images.
type ImageFlavor struct {
	// MainRegistry is a registry for all images except of collector.
	MainRegistry       string
	MainImageName      string
	MainImageTag       string
	CentralDBImageTag  string
	CentralDBImageName string

	// CollectorRegistry may be different from MainRegistry in case of stackrox.io.
	CollectorRegistry      string
	CollectorImageName     string
	CollectorImageTag      string
	CollectorSlimImageName string
	CollectorSlimImageTag  string

	// ScannerImageTag is used for all scanner* images (scanner, scanner-db, scanner-slim and scanner-db-slim)
	ScannerImageTag        string
	ScannerImageName       string
	ScannerSlimImageName   string
	ScannerDBImageName     string
	ScannerDBSlimImageName string

	ChartRepo        ChartRepo
	ImagePullSecrets ImagePullSecrets
	Versions         version.Versions
}

// DevelopmentBuildImageFlavor returns image values for `development_build` flavor.
func DevelopmentBuildImageFlavor() ImageFlavor {
	v := version.GetAllVersionsDevelopment()
	collectorTag := v.CollectorVersion + "-latest"
	collectorSlimName := "collector"
	collectorSlimTag := v.CollectorVersion + "-slim"
	if buildinfo.ReleaseBuild {
		v = version.GetAllVersionsUnified()
		collectorTag = v.CollectorVersion
		collectorSlimName = "collector-slim"
		collectorSlimTag = v.CollectorVersion
	}
	return ImageFlavor{
		MainRegistry:       "quay.io/rhacs-eng",
		MainImageName:      "main",
		MainImageTag:       v.MainVersion,
		CentralDBImageTag:  v.MainVersion,
		CentralDBImageName: "central-db",

		CollectorRegistry:      "quay.io/rhacs-eng",
		CollectorImageName:     "collector",
		CollectorImageTag:      collectorTag,
		CollectorSlimImageName: collectorSlimName,
		CollectorSlimImageTag:  collectorSlimTag,

		ScannerImageName:       "scanner",
		ScannerSlimImageName:   "scanner-slim",
		ScannerImageTag:        v.ScannerVersion,
		ScannerDBImageName:     "scanner-db",
		ScannerDBSlimImageName: "scanner-db-slim",

		ChartRepo: ChartRepo{
			URL:     "https://mirror.openshift.com/pub/rhacs/charts",
			IconURL: "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/Red_Hat-Hat_icon.png",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: true,
		},
		Versions: v,
	}
}

// StackRoxIOReleaseImageFlavor returns image values for `stackrox.io` flavor.
// TODO(ROX-11642): remove stackrox.io flavor as stackrox.io image distribution is shut down.
func StackRoxIOReleaseImageFlavor() ImageFlavor {
	v := version.GetAllVersionsUnified()
	return ImageFlavor{
		MainRegistry:       "stackrox.io",
		MainImageName:      "main",
		MainImageTag:       v.MainVersion,
		CentralDBImageTag:  v.MainVersion,
		CentralDBImageName: "central-db",

		CollectorRegistry:      "collector.stackrox.io",
		CollectorImageName:     "collector",
		CollectorImageTag:      v.CollectorVersion,
		CollectorSlimImageName: "collector-slim",
		CollectorSlimImageTag:  v.CollectorVersion,

		ScannerImageName:       "scanner",
		ScannerSlimImageName:   "scanner-slim",
		ScannerImageTag:        v.ScannerVersion,
		ScannerDBImageName:     "scanner-db",
		ScannerDBSlimImageName: "scanner-db-slim",

		ChartRepo: ChartRepo{
			URL:     "https://charts.stackrox.io",
			IconURL: "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/Red_Hat-Hat_icon.png",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: false,
		},
		Versions: v,
	}
}

// RHACSReleaseImageFlavor returns image values for `rhacs` flavor.
func RHACSReleaseImageFlavor() ImageFlavor {
	v := version.GetAllVersionsUnified()
	return ImageFlavor{
		MainRegistry:       "registry.redhat.io/advanced-cluster-security",
		MainImageName:      "rhacs-main-rhel8",
		MainImageTag:       v.MainVersion,
		CentralDBImageTag:  v.MainVersion,
		CentralDBImageName: "rhacs-central-db-rhel8",

		CollectorRegistry:      "registry.redhat.io/advanced-cluster-security",
		CollectorImageName:     "rhacs-collector-rhel8",
		CollectorImageTag:      v.CollectorVersion,
		CollectorSlimImageName: "rhacs-collector-slim-rhel8",
		CollectorSlimImageTag:  v.CollectorVersion,

		ScannerImageName:       "rhacs-scanner-rhel8",
		ScannerSlimImageName:   "rhacs-scanner-slim-rhel8",
		ScannerImageTag:        v.ScannerVersion,
		ScannerDBImageName:     "rhacs-scanner-db-rhel8",
		ScannerDBSlimImageName: "rhacs-scanner-db-slim-rhel8",

		ChartRepo: ChartRepo{
			URL:     "https://mirror.openshift.com/pub/rhacs/charts",
			IconURL: "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/Red_Hat-Hat_icon.png",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: true,
		},
		Versions: v,
	}
}

// OpenSourceImageFlavor returns image values for `opensource` flavor. Opensource flavor can be used both in development
// and in releases.
// In non-release builds, i.e. in development, Collector and Scanner should have original tags so that developers don't
// need to retag Collector and Scanner images every time they commit to this repo.
// Release builds get unified tags like in other release image flavors.
func OpenSourceImageFlavor() ImageFlavor {
	v := version.GetAllVersionsDevelopment()
	collectorTag := v.CollectorVersion + "-latest"
	collectorSlimName := "collector"
	collectorSlimTag := v.CollectorVersion + "-slim"
	if buildinfo.ReleaseBuild {
		v = version.GetAllVersionsUnified()
		collectorTag = v.CollectorVersion
		collectorSlimName = "collector-slim"
		collectorSlimTag = v.CollectorVersion
	}
	return ImageFlavor{
		MainRegistry:       "quay.io/stackrox-io",
		MainImageName:      "main",
		MainImageTag:       v.MainVersion,
		CentralDBImageTag:  v.MainVersion,
		CentralDBImageName: "central-db",

		CollectorRegistry:      "quay.io/stackrox-io",
		CollectorImageName:     "collector",
		CollectorImageTag:      collectorTag,
		CollectorSlimImageName: collectorSlimName,
		CollectorSlimImageTag:  collectorSlimTag,

		ScannerImageName:       "scanner",
		ScannerSlimImageName:   "scanner-slim",
		ScannerImageTag:        v.ScannerVersion,
		ScannerDBImageName:     "scanner-db",
		ScannerDBSlimImageName: "scanner-db-slim",

		ChartRepo: ChartRepo{
			URL:     "https://raw.githubusercontent.com/stackrox/helm-charts/main/opensource/",
			IconURL: "https://raw.githubusercontent.com/stackrox/stackrox/master/image/templates/helm/shared/assets/StackRox_icon.png",
		},
		ImagePullSecrets: ImagePullSecrets{
			AllowNone: true,
		},
		Versions: v,
	}
}

// GetVisibleImageFlavorNames returns a string slice with image flavor names that should be displayed to the user
// depending on whether the code was compiled in release mode or not. Note that hidden flavors can still be used.
func GetVisibleImageFlavorNames(isReleaseBuild bool) []string {
	result := make([]string, 0, len(allImageFlavors))
	for _, f := range allImageFlavors {
		if f.isVisibleInReleaseBuild || !isReleaseBuild {
			result = append(result, f.imageFlavorName)
		}
	}
	return result
}

// checkImageFlavorName returns error if the image flavor name is unknown (or empty), nil otherwise.
func checkImageFlavorName(imageFlavorName string, isReleaseBuild bool) error {
	if _, exists := imageFlavorMap[imageFlavorName]; exists {
		return nil
	}
	return errors.Errorf("unexpected value '%s', allowed values are %v", imageFlavorName, GetVisibleImageFlavorNames(isReleaseBuild))
}

// GetImageFlavorByName returns ImageFlavor struct created for the provided flavorName if the name is valid, otherwise
// it returns an error.
func GetImageFlavorByName(flavorName string, isReleaseBuild bool) (ImageFlavor, error) {
	if err := checkImageFlavorName(flavorName, isReleaseBuild); err != nil {
		return ImageFlavor{}, err
	}

	return getImageFlavorByName(flavorName), nil
}

// GetImageFlavorNameFromEnv returns the value of the environment variable (ROX_IMAGE_FLAVOR) providing
// development_build as the default if not a ReleaseBuild and the environment variable is not set.
// This function will panic if running a ReleaseBuild and ROX_IMAGE_FLAVOR is not available.
func GetImageFlavorNameFromEnv() string {
	envValue := strings.TrimSpace(imageFlavorEnv())
	if envValue == "" && !buildinfo.ReleaseBuild {
		envValue = ImageFlavorNameDevelopmentBuild
		log.Warnf("Environment variable %s not set, this will cause a panic in release build. Assuming this code is executed in unit test session and using '%s' as default", ImageFlavorEnvName, envValue)
	}
	err := checkImageFlavorName(envValue, buildinfo.ReleaseBuild)
	if err != nil {
		// Panic if environment variable's value is incorrect to loudly signal improper configuration of the effectively
		// build-time constant.
		panicImageFlavorEnv(err)
	}

	return envValue
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
	envValue := GetImageFlavorNameFromEnv()
	return getImageFlavorByName(envValue)
}

func getImageFlavorByName(name string) ImageFlavor {
	return imageFlavorMap[name].constructorFunc()
}

func panicImageFlavorEnv(err error) {
	log.Panicf("Incorrect image flavor in environment variable %s: %s", ImageFlavorEnvName, err)
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

// ScannerSlimImage is the container image reference (full name) for the scanner-slim image.
func (f *ImageFlavor) ScannerSlimImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.ScannerSlimImageName, f.ScannerImageTag)
}

// ScannerDBImage is the container image reference (full name) for the scanner-db image.
func (f *ImageFlavor) ScannerDBImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.ScannerDBImageName, f.ScannerImageTag)
}

// MainImage is the container image reference (full name) for the "main" image.
func (f *ImageFlavor) MainImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.MainImageName, f.MainImageTag)
}

// MainImageNoTag is the container image repository (image name including registry, excluding tag) for the "main" image.
func (f *ImageFlavor) MainImageNoTag() string {
	return fmt.Sprintf("%s/%s", f.MainRegistry, f.MainImageName)
}

// CentralDBImage is the container image reference (full name) for the central-db image.
func (f *ImageFlavor) CentralDBImage() string {
	return fmt.Sprintf("%s/%s:%s", f.MainRegistry, f.CentralDBImageName, f.CentralDBImageTag)
}

// CollectorFullImage is the container image reference (full name) for the "collector" image
func (f *ImageFlavor) CollectorFullImage() string {
	return fmt.Sprintf("%s/%s:%s", f.CollectorRegistry, f.CollectorImageName, f.CollectorImageTag)
}

// CollectorSlimImage is the container image reference (full name) for the "collector slim" image
func (f *ImageFlavor) CollectorSlimImage() string {
	return fmt.Sprintf("%s/%s:%s", f.CollectorRegistry, f.CollectorSlimImageName, f.CollectorSlimImageTag)
}

// CollectorSlimImageNoTag is the container image repository (image name including registry, excluding tag) for the "collector slim" image.
func (f *ImageFlavor) CollectorSlimImageNoTag() string {
	return fmt.Sprintf("%s/%s", f.CollectorRegistry, f.CollectorSlimImageName)
}

// CollectorFullImageNoTag is the container image repository (image name including registry, excluding tag) for the  "collector" image.
func (f *ImageFlavor) CollectorFullImageNoTag() string {
	return fmt.Sprintf("%s/%s", f.CollectorRegistry, f.CollectorImageName)
}
