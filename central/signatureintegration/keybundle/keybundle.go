package keybundle

import (
	"github.com/stackrox/rox/central/signatureintegration/store"
	"time"
)

var watcher *Watcher

// StartWatcher creates and starts a key bundle watcher if the interval is non-zero.
func StartWatcher(filePath string, interval time.Duration, siStore store.SignatureIntegrationStore) {
	if interval == 0 {
		log.Info("Red Hat signing key bundle watcher is disabled (interval=0)")
		return
	}
	w := NewWatcher(filePath, interval, siStore)
	w.Start()
	watcher = w
}

// GetWatcher returns the running watcher, or nil if not started.
func GetWatcher() *Watcher {
	return watcher
}
