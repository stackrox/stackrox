package datastore

import (
	"context"

	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	policyStore "github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/central/policy/store/boltdb"
	policyPostgres "github.com/stackrox/rox/central/policy/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var storage policyStore.Store
	var indexer index.Indexer

	if features.PostgresDatastore.Enabled() {
		storage = policyPostgres.New(context.TODO(), globaldb.GetPostgres())
		indexer = policyPostgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = boltdb.New(globaldb.GetGlobalDB())
		indexer = index.New(globalindex.GetGlobalTmpIndex())
	}
	searcher := search.New(storage, indexer)

	clusterDatastore := clusterDS.Singleton()
	notifierDatastore := notifierDS.Singleton()

	ad = New(storage, indexer, searcher, clusterDatastore, notifierDatastore)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
