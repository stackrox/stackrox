package enhancement

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	managerInstance     *Watcher
	managerInstanceInit sync.Once
)

// WatcherSingleton returns the singleton instance for the sensor enhanced deployment response.
func WatcherSingleton() *Watcher {
	managerInstanceInit.Do(func() {
		managerInstance = NewWatcher()
	})

	return managerInstance
}
