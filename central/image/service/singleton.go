package service

import (
	cveDataStore "github.com/stackrox/stackrox/central/cve/datastore"
	"github.com/stackrox/stackrox/central/enrichment"
	"github.com/stackrox/stackrox/central/image/datastore"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/stackrox/central/watchedimage/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), cveDataStore.Singleton(), watchedImageDataStore.Singleton(), manager.Singleton(), connection.ManagerSingleton(), enrichment.ImageEnricherSingleton(), enrichment.ImageMetadataCacheSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
