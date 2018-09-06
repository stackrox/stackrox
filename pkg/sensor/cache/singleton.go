package cache

import (
	"sync"
)

var (
	once sync.Once

	as *PendingEvents
)

func initialize() {
	// Caches
	as = newPendingEvents()
}

// Singleton implements a singleton for the client streaming gRPC service between collector and sensor
func Singleton() *PendingEvents {
	once.Do(initialize)
	return as
}
