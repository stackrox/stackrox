package streamer

import (
	"sync"

	"github.com/stackrox/rox/central/sensor/service/pipeline/all"
)

var (
	once sync.Once

	sm Manager
)

// ManagerSingleton provides the instance of the Manager interface to use for managing sensor event streams with
// multiple clusters.
func ManagerSingleton() Manager {
	once.Do(func() {
		sm = NewManager(all.Singleton())
	})
	return sm
}
