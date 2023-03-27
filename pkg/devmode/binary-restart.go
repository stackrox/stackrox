package devmode

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/osutils"
)

var (
	log = logging.LoggerForModule()
)

type restartHandler struct{}

func (r *restartHandler) OnChange(_ string) (interface{}, error) {
	osutils.Restart()
	return nil, nil
}

func (r *restartHandler) OnStableUpdate(_ interface{}, _ error) {}

func (r *restartHandler) OnWatchError(err error) {
	log.Error(err)
}

// startBinaryWatchdog will restart the container once the underlying binary has changed
func startBinaryWatchdog(path string) {
	opts := k8scfgwatch.Options{
		Interval: 5 * time.Second,
		Force:    false,
	}
	log.Infof("Starting watchdog on %q", path)
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), path, k8scfgwatch.DeduplicateWatchErrors(&restartHandler{}), opts)
}
