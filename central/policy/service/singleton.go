package service

import (
	"context"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/metrics"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/cache"
	mitreDataStore "github.com/stackrox/rox/pkg/mitre/datastore"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(),
		clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		networkPolicyDS.Singleton(),
		notifierDataStore.Singleton(),
		mitreDataStore.Singleton(),
		reprocessor.Singleton(),
		lifecycle.SingletonManager(),
		notifierProcessor.Singleton(),
		cache.ImageMetadataCacheSingleton(),
		connection.ManagerSingleton())

	count, _ := datastore.Singleton().Count(context.Background(), search.NewQueryBuilder().AddExactMatches(search.PolicySource, storage.PolicySource_DECLARATIVE.String()).ProtoQuery())
	metrics.UpdatePolicyAsCodeCRsReceivedGauge(count)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
