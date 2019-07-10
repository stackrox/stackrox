package devbuild

import (
	"strconv"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
)

var (
	setting = env.RegisterSetting("ROX_DEVELOPMENT_BUILD", env.WithDefault(strconv.FormatBool(!buildinfo.ReleaseBuild)))

	enabled = false
)

// IsEnabled whether this binary is running in dev build mode.
func IsEnabled() bool {
	return enabled
}
