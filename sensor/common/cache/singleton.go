package cache

import (
	"sync"
)

var (
	once sync.Once

	as *ContainerCache
)

func initialize() {
	// Caches
	as = newContainerCache()
}

// Singleton implements a singleton for the client streaming gRPC service between collector and sensor
func Singleton() *ContainerCache {
	once.Do(initialize)
	return as
}
