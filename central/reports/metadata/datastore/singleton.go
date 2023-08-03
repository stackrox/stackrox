package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/reports/metadata/datastore/search"
	pgStore "github.com/stackrox/rox/central/reports/metadata/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/env"
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

// Singleton returns a singleton instance of ReportMetadata datastore
func Singleton() DataStore {
	if !env.VulnReportingEnhancements.BooleanSetting() {
		return nil
	}
	once.Do(initialize)
	return ds
}
