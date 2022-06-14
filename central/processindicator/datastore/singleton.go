package datastore

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/processindicator/index"
	"github.com/stackrox/stackrox/central/processindicator/pruner"
	"github.com/stackrox/stackrox/central/processindicator/search"
	"github.com/stackrox/stackrox/central/processindicator/store"
	"github.com/stackrox/stackrox/central/processindicator/store/postgres"
	"github.com/stackrox/stackrox/central/processindicator/store/rocksdb"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

const (
	pruneInterval     = 10 * time.Minute
	minArgsPerProcess = 5
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	var storage store.Store
	var indexer index.Indexer
	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
	} else {
		storage = rocksdb.New(globaldb.GetRocksDB())
		indexer = index.New(globalindex.GetProcessIndex())
	}
	searcher := search.New(storage, indexer)

	p := pruner.NewFactory(minArgsPerProcess, pruneInterval)

	var err error
	ad, err = New(storage, indexer, searcher, p)
	utils.CrashOnError(errors.Wrap(err, "unable to load datastore for process indicators"))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
