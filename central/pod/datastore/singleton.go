package datastore

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	"github.com/stackrox/rox/central/pod/index"
	"github.com/stackrox/rox/central/pod/store"
	pgStore "github.com/stackrox/rox/central/pod/store/postgres"
	pgIndex "github.com/stackrox/rox/central/pod/index/postgres"
	"github.com/stackrox/rox/central/pod/store/rocksdb"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ps DataStore

	log = logging.LoggerForModule()
)

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(func() {
		var err error
		var storage store.Store
		var indexer index.Indexer
		if features.PostgresPOC.Enabled() {
			storage = pgStore.New(globaldb.GetPostgresDB())
			indexer = pgIndex.NewIndexer(globaldb.GetPostgresDB())
		} else {
			storage = rocksdb.New(globaldb.GetRocksDB())
			indexer = index.New(globalindex.GetPodIndex())
		}
		searcher := search.New(storage, indexer)
		ps, err = New(
			storage,
			indexer,
			searcher,
			piDS.Singleton(),
			filter.Singleton(),
		)
		utils.CrashOnError(errors.Wrap(err, "unable to load datastore for pods"))
	})
	return ps
}
