package manager

import (
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/cache"
)

var (
	once    sync.Once
	manager Manager
)

// newService creates a new streaming service with the collector. It should only be called once.
func newManager() Manager {
	return &networkFlowManager{
		done:                concurrency.NewSignal(),
		connectionsByHost:   make(map[string]*hostConnections),
		pendingCache:        cache.Singleton(),
		enrichedConnections: make(map[networkConnIndicator]time.Time),
	}
}

func initialize() {
	// Creates the signal service with the pending cache embedded
	manager = newManager()
}

// Singleton implements a singleton for a network flow manager
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
