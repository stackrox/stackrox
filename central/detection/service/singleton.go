package service

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	networkpolicyDatastore "github.com/stackrox/rox/central/networkpolicies/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/risk/manager"
	sacHelper "github.com/stackrox/rox/central/sac/helper"
	"github.com/stackrox/rox/central/sensor/enhancement"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	clusterDS := clusterDatastore.Singleton()

	as = New(
		clusterDS,
		enrichment.ImageEnricherSingleton(),
		imageDatastore.Singleton(),
		manager.Singleton(),
		enrichment.Singleton(),
		buildTimeDetection.SingletonDetector(),
		processor.Singleton(),
		deploytime.SingletonDetector(),
		deploytime.SingletonPolicySet(),
		sacHelper.NewClusterSacHelper(clusterDS),
		connection.ManagerSingleton(),
		enhancement.BrokerSingleton(),
		networkpolicyDatastore.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
