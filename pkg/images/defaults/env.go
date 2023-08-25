package defaults

import "github.com/stackrox/rox/pkg/env"

const (
	// ImageFlavorEnvName is the name of the environment variable that controls the effective image flavor.
	ImageFlavorEnvName = "ROX_IMAGE_FLAVOR"

	// ImageFlavorNameDevelopmentBuild is a name for image flavor (image defaults) for images released to
	// quay.io/rhacs-eng for internal use by the Red Hat development team.
	// Note that release or non-release compilation of Go code is determined separately from this image flavor and
	// unrelated to the image flavor. It is possible that binaries compiled in release mode are packaged in images that
	// have development_build image flavor.
	ImageFlavorNameDevelopmentBuild = "development_build"
	// ImageFlavorNameStackRoxIORelease is a name for image flavor (image defaults) for images released to stackrox.io.
	ImageFlavorNameStackRoxIORelease = "stackrox.io"
	// ImageFlavorNameRHACSRelease is a name for image flavor (image defaults) for images released to registry.redhat.io.
	ImageFlavorNameRHACSRelease = "rhacs"
	// ImageFlavorNameOpenSource is a name for image flavor (image defaults) for images released to quay.io/stackrox-io.
	ImageFlavorNameOpenSource = "opensource"
)

var (
	imageFlavorSetting = env.RegisterSetting(ImageFlavorEnvName)
)

// imageFlavorEnv returns the environment variable ROX_IMAGE_FLAVOR value
func imageFlavorEnv() string {
	return imageFlavorSetting.Setting()
}
