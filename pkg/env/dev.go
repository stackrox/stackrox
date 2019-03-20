package env

import (
	"strconv"

	"github.com/stackrox/rox/pkg/buildinfo"
)

var (
	// DevelopmentBuild signifies that we are in a development environment
	DevelopmentBuild = RegisterSetting("ROX_DEVELOPMENT_BUILD", WithDefault(strconv.FormatBool(!buildinfo.ReleaseBuild)))
)
