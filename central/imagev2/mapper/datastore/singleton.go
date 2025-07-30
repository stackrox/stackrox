package datastore

import (
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad imageDatastore.DataStore
)

func initialize() {
	ad = New()
}

// Singleton provides the interface for non-service external interaction.
func Singleton() imageDatastore.DataStore {
	once.Do(initialize)
	return ad
}
