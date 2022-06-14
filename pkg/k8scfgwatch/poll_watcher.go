package k8scfgwatch

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

const (
	stabilizationGracePeriod = 50 * time.Millisecond
)

type pollWatcher struct {
	dir     string
	handler Handler

	lastContentsMTime time.Time
}

func newPollWatcher(dir string, handler Handler) *pollWatcher {
	w := &pollWatcher{
		dir:     dir,
		handler: handler,
	}
	return w
}

func (w *pollWatcher) Init() error {
	return w.pollAndHandleOnce()
}

func (w *pollWatcher) Run(ctx concurrency.Waitable, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.pollAndHandleOnce(); err != nil {
				w.handler.OnWatchError(err)
			}
		}
	}
}

func (w *pollWatcher) pollAndHandleOnce() error {
	var val interface{}
	var updateErr error

	changed := false

	// Read the directory contents, and if the timestamps indicate a change, call the `OnChange` method of the handler
	// to process the updated directory contents. In all cases except for a failure to list directory entries, the
	// outcome (i.e., the result of `OnChange`) is not committed (via `OnStableUpdate`) until another iteration of this
	// loop indicates that there have been no further changes to the directory. This serves to prevent spurious errors
	// from being reported that might occur if, e.g., several files are being changed in a non-atomic way.
	for {
		contentsMTime, err := dirContentsMTime(w.dir)
		if err != nil {
			return errors.Wrapf(err, "reading contents of directory %s", w.dir)
		}

		if w.lastContentsMTime.Equal(contentsMTime) {
			break // stabilized
		}

		w.lastContentsMTime = contentsMTime

		val, updateErr = w.handler.OnChange(w.dir)
		changed = true

		time.Sleep(stabilizationGracePeriod)
	}

	if changed {
		w.handler.OnStableUpdate(val, updateErr)
	}
	return nil
}
