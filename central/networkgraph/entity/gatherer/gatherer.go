package gatherer

import (
	blobstore "github.com/stackrox/rox/central/blob/datastore"
	entityDataStore "github.com/stackrox/rox/central/networkgraph/entity/datastore"
)

// NetworkGraphDefaultExtSrcsGatherer provides functionality to update the storage.NetworkEntity storage with default network graph
// external sources.
type NetworkGraphDefaultExtSrcsGatherer interface {
	Start()
	Stop()

	Update() error
}

// NewNetworkGraphDefaultExtSrcsGatherer returns an instance of NetworkGraphDefaultExtSrcsGatherer as per the offline mode setting.
func NewNetworkGraphDefaultExtSrcsGatherer(networkEntityDS entityDataStore.EntityDataStore, blobStore blobstore.Datastore) NetworkGraphDefaultExtSrcsGatherer {
	return newDefaultExtNetworksGatherer(networkEntityDS, blobStore)
}
