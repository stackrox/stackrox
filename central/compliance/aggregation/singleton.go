package aggregation

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/compliance/standards"
	complianceStore "github.com/stackrox/rox/central/compliance/store"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeStore "github.com/stackrox/rox/central/node/globalstore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ag   Aggregator
)

func initialize() {
	ag = New(
		complianceStore.Singleton(),
		standards.RegistrySingleton(),
		clusterDatastore.Singleton(),
		namespaceStore.Singleton(),
		nodeStore.Singleton(),
	)
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() Aggregator {
	once.Do(initialize)
	return ag
}
