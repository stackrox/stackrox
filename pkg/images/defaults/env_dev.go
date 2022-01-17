//go:build !release
// +build !release

package defaults

import (
	"github.com/stackrox/rox/pkg/env"
)

var (
	// We set the default value to the image flavor to imageFlavorDevelopment.
	// This is done since some tests that depend on this package will fail when run locally if the environment variable
	// ROX_IMAGE_FLAVOR is not set.
	// Please notice that if ROX_IMAGE_FLAVOR is set to an empty value the tests will fail since an empty value
	// is respected but is not valid.
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName, env.AllowEmpty(), env.WithDefault(imageFlavorDevelopment))
)
