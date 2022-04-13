package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/alert/datastore/internal/commentsstore"
	"github.com/stackrox/stackrox/central/alert/datastore/internal/index"
	"github.com/stackrox/stackrox/central/alert/datastore/internal/search"
	"github.com/stackrox/stackrox/central/alert/datastore/internal/store"
	"github.com/stackrox/stackrox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/stackrox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once         sync.Once
	soleInstance DataStore
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer

	if features.PostgresDatastore.Enabled() {
		storage = store.NewFullStore(postgres.New(context.TODO(), globaldb.GetPostgres()))
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = store.NewFullStore(rocksdb.New(globaldb.GetRocksDB()))
		indexer = index.New(globalindex.GetAlertIndex())
	}
	commentsStorage := commentsstore.New(globaldb.GetGlobalDB())
	searcher := search.New(storage, indexer)
	var err error
	soleInstance, err = New(storage, commentsStorage, indexer, searcher)
	utils.CrashOnError(errors.Wrap(err, "unable to load datastore for alerts"))
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(initialize)
	return soleInstance
}
