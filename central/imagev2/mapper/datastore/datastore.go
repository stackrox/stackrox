package datastore

import (
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
)

// NewWithPostgres returns a new instance of DataStore using the input store, and searcher.
func New() imageDatastore.DataStore {
	ds := newDatastoreImpl(imageDatastore.Singleton(), imageV2Datastore.Singleton())
	return ds
}
