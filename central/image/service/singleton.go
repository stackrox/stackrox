package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	sacHelper "github.com/stackrox/rox/central/sac/helper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	"github.com/stackrox/rox/pkg/images/cache"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		datastore.Singleton(),
		watchedImageDataStore.Singleton(),
		manager.Singleton(),
		connection.ManagerSingleton(),
		enrichment.ImageEnricherSingleton(),
		cache.ImageMetadataCacheSingleton(),
		scanwaiter.Singleton(),
		sacHelper.NewClusterSacHelper(clusterDataStore.Singleton()),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
