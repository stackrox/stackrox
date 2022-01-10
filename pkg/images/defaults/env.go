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
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName)
)

// imageFlavorEnv returns the environment variable ROX_IMAGE_FLAVOR value
func imageFlavorEnv() string {
	return imageFlavorSetting.Setting()
}
