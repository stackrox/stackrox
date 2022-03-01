package datastore

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/internal/commentsstore"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store/rocksdb"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
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
	storage := rocksdb.New(globaldb.GetRocksDB())
	commentsStorage := commentsstore.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetProcessIndex())
	searcher := search.New(storage, indexer)

	p := pruner.NewFactory(minArgsPerProcess, pruneInterval)

	var err error
	ad, err = New(context.TODO(), storage, commentsStorage, indexer, searcher, p)
	utils.CrashOnError(errors.Wrap(err, "unable to load datastore for process indicators"))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
