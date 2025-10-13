package aggregation

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ag   Aggregator
)

func initialize() {
	ag = New(
		complianceDS.Singleton(),
		standards.RegistrySingleton(),
		clusterDatastore.Singleton(),
		namespaceStore.Singleton(),
		nodeDatastore.Singleton(),
		deploymentDatastore.Singleton(),
	)
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() Aggregator {
	once.Do(initialize)
	return ag
}
