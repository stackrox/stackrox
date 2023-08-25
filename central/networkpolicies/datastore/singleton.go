package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/search"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/store/postgres"
	undoPGStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undodeploymentstore/postgres"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	undoDeploymentStorage := undoPGStore.New(globaldb.GetPostgres())
	networkPolicyStorage := postgres.New(globaldb.GetPostgres())
	networkPolicySearcher := search.New(postgres.NewIndexer(globaldb.GetPostgres()))

	as = New(networkPolicyStorage, networkPolicySearcher, undostore.Singleton(), undoDeploymentStorage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
