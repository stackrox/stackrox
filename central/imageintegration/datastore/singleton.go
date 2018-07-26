package datastore

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/globaldb"
	"bitbucket.org/stack-rox/apollo/central/imageintegration"
	"bitbucket.org/stack-rox/apollo/central/imageintegration/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

var (
	once sync.Once

	storage store.Store

	ad DataStore
)

func initialize() {
	// Create underlying store and datastore.
	storage = store.New(globaldb.GetGlobalDB())
	ad = New(storage)

	// Initialize the integration set with all present integrations.
	toNotifyOfIntegrations := imageintegration.ToNotify()
	integrations, err := ad.GetImageIntegrations(&v1.GetImageIntegrationsRequest{})
	if err != nil {
		log.Errorf("unable to use previous integrations", err)
	}
	for _, ii := range integrations {
		if err := toNotifyOfIntegrations.NotifyUpdated(ii); err != nil {
			log.Errorf("unable to use previous integration", err)
		}
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
