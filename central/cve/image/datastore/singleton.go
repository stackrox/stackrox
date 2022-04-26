package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/image/datastore/internal/search"
	"github.com/stackrox/rox/central/cve/image/datastore/internal/store/postgres"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := postgres.New(context.TODO(), globaldb.GetPostgres())
	searcher := search.New(storage, cveIndexer.New(globalindex.GetGlobalIndex()))

	var err error
	ds, err = New(storage, cveIndexer.New(globalindex.GetGlobalIndex()), searcher)
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
