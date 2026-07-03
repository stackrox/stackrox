// Package watch detects changes to the local RPM package database so
// roxagent can rescan sooner than its periodic interval allows.
package watch

import (
	"errors"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/internal/hostprobe"
)

// candidateRPMDirs are RPM database directories to watch for package-change
// signals, in preference order (modern layout first). We watch the
// directories, not a specific DB file: RPM transactions commit via
// create/rename/write of WAL or temp files rather than a single in-place
// write, so watching the containing directory reliably catches all of these
// (the fsnotify-recommended pattern for this kind of change detection).
var candidateRPMDirs = []string{
	"/usr/lib/sysimage/rpm", // rpm >= 4.16, sqlite backend (RHEL 9+/Fedora)
	"/var/lib/rpm",          // older layout (RHEL 7/8), or a symlink to the above
}

// Watcher signals when the RPM package database changes on disk.
type Watcher struct {
	fsw       *fsnotify.Watcher
	triggerCh chan struct{}
}

// New starts watching the first existing, watchable RPM database directory
// under hostPath. Returns an error if none of the candidate directories
// could be watched; callers should treat this as best-effort and fall back
// to periodic-only scanning rather than failing startup.
func New(hostPath string) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	dir, err := addFirstWatchableDir(fsw, hostPath, candidateRPMDirs)
	if err != nil {
		_ = fsw.Close()
		return nil, err
	}
	log.Infof("Watching %q for reactive RPM/DNF change detection", dir)

	w := &Watcher{
		fsw:       fsw,
		triggerCh: make(chan struct{}, 1),
	}
	go w.run()
	return w, nil
}

func addFirstWatchableDir(fsw *fsnotify.Watcher, hostPath string, dirs []string) (string, error) {
	var errs []error
	for _, dir := range dirs {
		path := hostprobe.HostPathFor(hostPath, dir)
		if err := fsw.Add(path); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", path, err))
			continue
		}
		return path, nil
	}
	return "", fmt.Errorf("no watchable RPM database directory found: %w", errors.Join(errs...))
}

func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if !isRelevant(event) {
				continue
			}
			// Non-blocking send: the first event in a burst queues a
			// trigger; every subsequent event while it's still pending is
			// dropped, collapsing a whole RPM transaction's writes into one
			// pending rescan. This is trivial coalescing, not a debounce —
			// see the design doc for why that's an accepted Phase 1
			// limitation (a rescan can start mid-transaction; the next
			// trigger picks up the final state).
			select {
			case w.triggerCh <- struct{}{}:
			default:
			}
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Warnf("RPM database watcher error: %v", err)
		}
	}
}

func isRelevant(event fsnotify.Event) bool {
	return event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename)
}

// Triggered returns a channel that receives a value shortly after a change
// is detected in the watched RPM database directory. Multiple changes before
// the channel is drained collapse into a single pending trigger.
func (w *Watcher) Triggered() <-chan struct{} {
	return w.triggerCh
}

// Close stops the underlying fsnotify watcher.
func (w *Watcher) Close() error {
	return w.fsw.Close()
}
