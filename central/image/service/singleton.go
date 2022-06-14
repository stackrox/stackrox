package service

import (
	legacyImageCVEDataStore "github.com/stackrox/stackrox/central/cve/datastore"
	cveDataStore "github.com/stackrox/stackrox/central/cve/image/datastore"
	"github.com/stackrox/stackrox/central/enrichment"
	"github.com/stackrox/stackrox/central/image/datastore"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/stackrox/central/watchedimage/datastore"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	var imageCVEDataStore cveDataStore.DataStore
	if features.PostgresDatastore.Enabled() {
		imageCVEDataStore = cveDataStore.Singleton()
	} else {
		imageCVEDataStore = legacyImageCVEDataStore.Singleton()
	}
	as = New(datastore.Singleton(), imageCVEDataStore, watchedImageDataStore.Singleton(), manager.Singleton(), connection.ManagerSingleton(), enrichment.ImageEnricherSingleton(), enrichment.ImageMetadataCacheSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
