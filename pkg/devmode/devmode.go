package devmode

import (
	"path/filepath"
	"runtime"

	"github.com/stackrox/stackrox/pkg/buildinfo"
	"github.com/stackrox/stackrox/pkg/debughandler"
	"github.com/stackrox/stackrox/pkg/devbuild"
	"github.com/stackrox/stackrox/pkg/env"
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
