package aggregation

import (
	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	complianceDS "github.com/stackrox/stackrox/central/compliance/datastore"
	"github.com/stackrox/stackrox/central/compliance/standards"
	deploymentDatastore "github.com/stackrox/stackrox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/stackrox/central/namespace/datastore"
	nodeDatastore "github.com/stackrox/stackrox/central/node/globaldatastore"
	"github.com/stackrox/stackrox/pkg/sync"
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
