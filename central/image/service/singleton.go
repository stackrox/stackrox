package service

import (
	biDataStore "github.com/stackrox/rox/central/baseimage/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/image/datastore"
	imageMapperDatastore "github.com/stackrox/rox/central/imagev2/datastore/mapper/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
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
		imageMapperDatastore.Singleton(),
		watchedImageDataStore.Singleton(),
		manager.Singleton(),
		connection.ManagerSingleton(),
		enrichment.ImageEnricherSingleton(),
		cache.ImageMetadataCacheSingleton(),
		scanwaiter.Singleton(),
		sachelper.NewClusterSacHelper(clusterDataStore.Singleton()),
		biDataStore.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
