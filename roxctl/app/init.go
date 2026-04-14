package app

import (
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/helm"
)

// initComponentLogic initializes all roxctl-specific components that were
// previously using init() functions.
func initComponentLogic() {
	initImageFlavorDefaults()
	helm.Init()
}

// initImageFlavorDefaults initializes the default image flavor based on build type.
// This was previously done in roxctl/common/flags/imageFlavor.go init().
func initImageFlavorDefaults() {
	if !buildinfo.ReleaseBuild {
		// Use the string constant directly to avoid importing pkg/images/defaults
		// which would create an import cycle
		flags.SetImageFlavorDefault("development_build")
	}
}
