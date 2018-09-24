package streamer

import (
	"sync"

	"github.com/stackrox/rox/central/sensorevent/service/pipeline/all"
	"github.com/stackrox/rox/central/sensorevent/store"
)

var (
	once sync.Once

	sm Manager
)

// ManagerSingleton provides the instance of the Manager interface to use for managing sensor event streams with
// multiple clusters.
func ManagerSingleton() Manager {
	once.Do(func() {
		sm = NewManager(store.Singleton(), all.Singleton())
	})
	return sm
}
