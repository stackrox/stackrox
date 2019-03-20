package manager

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/clusterentities"
)

var (
	once    sync.Once
	manager Manager
)

// newService creates a new streaming service with the collector. It should only be called once.
func newManager() Manager {
	return &networkFlowManager{
		done:              concurrency.NewSignal(),
		connectionsByHost: make(map[string]*hostConnections),
		clusterEntities:   clusterentities.StoreInstance(),
		flowUpdates:       make(chan *central.NetworkFlowUpdate),
	}
}

func initialize() {
	// Creates the signal service
	manager = newManager()
}

// Singleton implements a singleton for a network flow manager
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
