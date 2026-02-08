// This package resolves import cycle between datastores and the runner.
package refresh

import (
	"github.com/stackrox/rox/pkg/sync"
)

type Refresher interface {
	RefreshTracker(prefix string)
}

var (
	singleton Refresher
	mu        sync.RWMutex
)

// SetSingleton initializes the refresher singleton object.
func SetSingleton(r Refresher) {
	mu.Lock()
	defer mu.Unlock()
	singleton = r
}

// RefreshTracker refreshes the tracker, identified by the metric prefix.
func RefreshTracker(prefix string) {
	mu.RLock()
	defer mu.RUnlock()
	if singleton != nil {
		singleton.RefreshTracker(prefix)
	}
}
