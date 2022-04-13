package defaults

import "github.com/stackrox/rox/pkg/env"

const (
	// ImageFlavorEnvName is the variable storing the image flavor name for different builds.
	ImageFlavorEnvName = "ROX_IMAGE_FLAVOR"

	// ImageFlavorNameDevelopmentBuild is a name for image flavor (image defaults) for development builds.
	ImageFlavorNameDevelopmentBuild = "development_build"
	// ImageFlavorNameStackRoxIORelease is a name for image flavor (image defaults) for images released to stackrox.io.
	ImageFlavorNameStackRoxIORelease = "stackrox.io"
	// ImageFlavorNameRHACSRelease is a name for image flavor (image defaults) for images released to registry.redhat.io.
	ImageFlavorNameRHACSRelease = "rhacs"
)

var (
	imageFlavorSetting = env.RegisterSetting(ImageFlavorEnvName)
)

// ImageFlavorEnv returns the environment variable ROX_IMAGE_FLAVOR value
func ImageFlavorEnv() string {
	return imageFlavorSetting.Setting()
}
