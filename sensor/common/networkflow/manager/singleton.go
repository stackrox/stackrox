package manager

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/externalsrcs"
)

var (
	once    sync.Once
	manager Manager
)

// newService creates a new streaming service with the collector. It should only be called once.
func newManager(clusterEntities *clusterentities.Store, externalSrcs externalsrcs.Store) Manager {
	mgr := &networkFlowManager{
		done:              concurrency.NewSignal(),
		connectionsByHost: make(map[string]*hostConnections),
		clusterEntities:   clusterEntities,
		flowUpdates:       make(chan *central.MsgFromSensor),
		publicIPs:         newPublicIPsManager(),
		externalSrcs:      externalSrcs,
	}

	return mgr
}

func initialize() {
	// Creates the signal service
	manager = newManager(clusterentities.StoreInstance(), externalsrcs.StoreInstance())
}

// Singleton implements a singleton for a network flow manager
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
