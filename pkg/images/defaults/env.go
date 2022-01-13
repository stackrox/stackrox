package defaults

import "github.com/stackrox/rox/pkg/env"

const (
	imageFlavorEnvName = "ROX_IMAGE_FLAVOR"

	// ImageFlavorNameDevelopmentBuild is a name for image flavor (image defaults) for development builds.
	ImageFlavorNameDevelopmentBuild = "development"
	// ImageFlavorNameStackRoxIORelease is a name for image flavor (image defaults) for images released to stackrox.io.
	ImageFlavorNameStackRoxIORelease = "stackrox.io"
	// TODO(RS-380): add this flavor:
	// ImageFlavorNameRHACSRelease       = "rhacs"
)

var (
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName)
)

// imageFlavorEnv returns the environment variable ROX_IMAGE_FLAVOR value
func imageFlavorEnv() string {
	return imageFlavorSetting.Setting()
}
