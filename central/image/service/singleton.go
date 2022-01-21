package service

import (
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), cveDataStore.Singleton(), watchedImageDataStore.Singleton(), manager.Singleton(), enrichment.ImageEnricherSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
