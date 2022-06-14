package devmode

import (
	"path/filepath"
	"runtime"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/debughandler"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/env"
)

const binaryRoot = "/stackrox"

// StartOnDevBuilds start the development mode only if a dev build is enabled.
// Enables a binary watchdog to restart the container if the underlying binary changed.
func StartOnDevBuilds(binaryPath string) {
	if !devbuild.IsEnabled() || buildinfo.ReleaseBuild {
		return
	}

	if env.HotReload.BooleanSetting() {
		log.Warn("")
		log.Warn("***********************************************************************************")
		log.Warn("This binary is being hot reloaded. It may be a different version from the image tag")
		log.Warn("***********************************************************************************")
		log.Warn("***********************************************************************************")

		go startBinaryWatchdog(filepath.Join(binaryRoot, binaryPath))
	}

	debughandler.MustStartServerAsync("")

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
}
