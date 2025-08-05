package datastore

import (
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad imageDatastore.DataStore
)

func initialize() {
	ad = New(imageDatastore.Singleton(), imageV2Datastore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() imageDatastore.DataStore {
	once.Do(initialize)
	return ad
}
