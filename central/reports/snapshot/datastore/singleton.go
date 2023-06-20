package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/reports/snapshot/datastore/search"
	pgStore "github.com/stackrox/rox/central/reports/snapshot/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once
	ds   DataStore
)

func initialize() {
	var err error
	storage := pgStore.New(globaldb.GetPostgres())
	indexer := pgStore.NewIndexer(globaldb.GetPostgres())
	ds, err = New(storage, search.New(storage, indexer))
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of ReportSnapshot datastore
func Singleton() DataStore {
	if !features.VulnMgmtReportingEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ds
}
