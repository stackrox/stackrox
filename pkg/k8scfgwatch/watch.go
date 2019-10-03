package k8scfgwatch

import (
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	// DefaultInterval is the default polling interval for a config mount watcher.
	DefaultInterval = 30 * time.Second
)

// Options controls the behavior of watching a config mount directory.
type Options struct {
	Interval time.Duration
	Force    bool
}

func (o *Options) interval() time.Duration {
	if o.Interval != 0 {
		return o.Interval
	}
	return DefaultInterval
}

// WatchConfigMountDir watches the given directory in the background, bound by the given context. Whenever a change is
// detected, the handler is invoked with the respective method.
// Upon initialization, the initial data is attempted to be read, and if successfully, the handler (both for `OnChange`
// and `OnStableUpdate`) is invoked inline(!). If there is a watch error (e.g., directory does not exist or cannot be
// read) and the force option is set to false, this error is returned and the watch is not started. Otherwise, with
// force set to true, the `OnWatchError` method of the handler is invoked with the error, and the
// watch attempts will continue as long as the context is alive.
func WatchConfigMountDir(ctx concurrency.Waitable, dir string, handler Handler, opts Options) error {
	w := newPollWatcher(dir, handler)
	if err := w.Init(); err != nil {
		if !opts.Force {
			return err
		}
		handler.OnWatchError(err)
	}
	go w.Run(ctx, opts.interval())
	return nil
}
