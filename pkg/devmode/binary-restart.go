package devmode

import (
	"context"
	"path/filepath"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/osutils"
)

const rootPath = "/stackrox/"

var (
	log = logging.LoggerForModule()
)

type restartHandler struct{}

func (r *restartHandler) OnChange(dir string) (interface{}, error) {
	osutils.Restart()
	return nil, nil
}

func (r *restartHandler) OnStableUpdate(val interface{}, err error) {}

func (r *restartHandler) OnWatchError(err error) {
	log.Error(err)
}

func startBinaryWatchdog(path string) {
	opts := k8scfgwatch.Options{
		Interval: 5 * time.Second,
		Force:    false,
	}
	log.Infof("Starting watchdog on %q", path)
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), path, k8scfgwatch.DeduplicateWatchErrors(&restartHandler{}), opts)
}

// StartBinaryWatchdog will restart the container once the underlying binary has changed
func StartBinaryWatchdog(binaryName string) {
	if buildinfo.ReleaseBuild {
		return
	}
	go startBinaryWatchdog(filepath.Join(rootPath, binaryName))
}
