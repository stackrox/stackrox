package datastore

import (
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
)

// New returns a new instance of DataStore.
func New(datastoreV1 imageDatastore.DataStore, datastoreV2 imageV2Datastore.DataStore) imageDatastore.DataStore {
	return newDatastoreImpl(datastoreV1, datastoreV2)
}
