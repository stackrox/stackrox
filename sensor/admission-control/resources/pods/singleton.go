package pods

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	podStore *PodStore
)

func initialize() {
	podStore = NewPodStore()
}

// Singleton provides the interface for getting annotation values with a datastore backed implementation.
func Singleton() *PodStore {
	once.Do(initialize)
	return podStore
}
