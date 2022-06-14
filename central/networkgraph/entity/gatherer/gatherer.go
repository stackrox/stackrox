package gatherer

import (
	entityDataStore "github.com/stackrox/stackrox/central/networkgraph/entity/datastore"
)

// NetworkGraphDefaultExtSrcsGatherer provides functionality to update the storage.NetworkEntity storage with default network graph
// external sources.
type NetworkGraphDefaultExtSrcsGatherer interface {
	Start()
	Stop()

	Update() error
}

// NewNetworkGraphDefaultExtSrcsGatherer returns an instance of NetworkGraphDefaultExtSrcsGatherer as per the offline mode setting.
func NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDS entityDataStore.EntityDataStore) NetworkGraphDefaultExtSrcsGatherer {
	return newDefaultExtNetworksGatherer(networkEntityDS)
}
