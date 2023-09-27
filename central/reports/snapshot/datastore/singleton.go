package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/reports/snapshot/datastore/search"
	pgStore "github.com/stackrox/rox/central/reports/snapshot/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ds   DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())
	ds = New(storage, search.New(storage, indexer))
}

// Singleton returns a singleton instance of ReportSnapshot datastore
func Singleton() DataStore {
	if !features.VulnReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ds
}
