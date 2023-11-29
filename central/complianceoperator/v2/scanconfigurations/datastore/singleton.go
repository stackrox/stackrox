package datastore

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/search"
	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())
	storage := pgStore.New(globaldb.GetPostgres())
	searchScanConfig := search.New(storage, indexer)
	dataStore = New(storage, statusStore.New(globaldb.GetPostgres()), clusterDatastore.Singleton(), searchScanConfig)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}
