package defaults

import "github.com/stackrox/rox/pkg/env"

const (
	imageFlavorEnvName = "ROX_IMAGE_FLAVOR"

	imageFlavorDevelopment = "development_build"
	imageFlavorStackroxIO  = "stackrox_io_release"
	// TODO(RS-380): add this flavor:
	// imageFlavorRHACS       = "rhacs_release"
)

var (
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName, env.WithDefault(imageFlavorStackroxIO))
)

// ImageFlavorFromEnv returns the environment variable ROX_IMAGE_FLAVOR value
func ImageFlavorFromEnv() string {
	return imageFlavorSetting.Setting()
}
