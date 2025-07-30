package datastore

import (
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
)

// New returns a new instance of DataStore.
func New() imageDatastore.DataStore {
	return newDatastoreImpl(imageDatastore.Singleton(), imageV2Datastore.Singleton())
}
