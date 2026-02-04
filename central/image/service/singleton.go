package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiterv2"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/image/datastore"
	imageV2Datastore "github.com/stackrox/rox/central/imagev2/datastore"
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
		imageV2Datastore.Singleton(),
		imageMapperDatastore.Singleton(),
		watchedImageDataStore.Singleton(),
		manager.Singleton(),
		connection.ManagerSingleton(),
		enrichment.ImageEnricherSingleton(),
		enrichment.ImageEnricherV2Singleton(),
		cache.ImageMetadataCacheSingleton(),
		scanwaiter.Singleton(),
		scanwaiterv2.Singleton(),
		sachelper.NewClusterSacHelper(clusterDataStore.Singleton()),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
