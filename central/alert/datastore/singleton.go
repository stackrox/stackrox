package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/alert/datastore/internal/commentsstore"
	"github.com/stackrox/rox/central/alert/datastore/internal/index"
	"github.com/stackrox/rox/central/alert/datastore/internal/search"
	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	pgIndex "github.com/stackrox/rox/central/alert/datastore/internal/index/postgres"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once         sync.Once
	soleInstance DataStore
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer
	if features.PostgresPOC.Enabled() {
		storage = pgStore.NewFullStore(globaldb.GetPostgresDB())
		indexer = pgIndex.NewIndexer(globaldb.GetPostgresDB())
	} else {
		storage = rocksdb.NewFullStore(globaldb.GetRocksDB())
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
