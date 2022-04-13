package devbuild

import (
	"strings"

	"github.com/stackrox/stackrox/pkg/buildinfo"
)

func init() {
	// Force enabled to false on release builds.
	enabled = strings.ToLower(setting.Setting()) == "true" && !buildinfo.ReleaseBuild

	// Panic if there is an inconsistency (impossible given the above line, but just in case somebody tries to be too
	// smart somewhere).
	if enabled && buildinfo.ReleaseBuild {
		panic("DEV BUILD SETTING IS ACTIVE IN A RELEASE BUILD. THIS SHOULD NEVER HAPPEN.")
	}
}
