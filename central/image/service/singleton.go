package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
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
		enrichment.ImageMetadataCacheSingleton(),
		scanwaiter.Singleton(),
		sachelper.NewClusterSacHelper(clusterDataStore.Singleton()),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
