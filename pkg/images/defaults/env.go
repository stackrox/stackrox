package defaults

import "github.com/stackrox/stackrox/pkg/env"

const (
	imageFlavorEnvName = "ROX_IMAGE_FLAVOR"

	// ImageFlavorNameDevelopmentBuild is a name for image flavor (image defaults) for development builds.
	ImageFlavorNameDevelopmentBuild = "development_build"
	// ImageFlavorNameStackRoxIORelease is a name for image flavor (image defaults) for images released to stackrox.io.
	ImageFlavorNameStackRoxIORelease = "stackrox.io"
	// ImageFlavorNameRHACSRelease is a name for image flavor (image defaults) for images released to registry.redhat.io.
	ImageFlavorNameRHACSRelease = "rhacs"
	// ImageFlavorNameOpenSource is a name for image flavor (image defaults) for images released to quay.io/stackrox-io.
	ImageFlavorNameOpenSource = "opensource"
)

var (
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName)
)

// imageFlavorEnv returns the environment variable ROX_IMAGE_FLAVOR value
func imageFlavorEnv() string {
	return imageFlavorSetting.Setting()
}
