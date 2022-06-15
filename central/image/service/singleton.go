package service

import (
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		return
	}
	imageCVEDataStore := cveDataStore.Singleton()
	as = New(datastore.Singleton(), imageCVEDataStore, watchedImageDataStore.Singleton(), manager.Singleton(), connection.ManagerSingleton(), enrichment.ImageEnricherSingleton(), enrichment.ImageMetadataCacheSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
