//go:build release
// +build release

package defaults

import (
	"github.com/stackrox/rox/pkg/env"
)

var (
	// We do not set a default value for the image flavor in a release build since there is no default value.
	// The environment variable ROX_IMAGE_FLAVOR should be explicitly set before or during the build time.
	imageFlavorSetting = env.RegisterSetting(imageFlavorEnvName)
)
