package datastore

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	pgStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
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
	storage := pgStore.New(globaldb.GetPostgres())
	plopStorage := plopStore.New(globaldb.GetPostgres())
	searcher := search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres()))

	p := pruner.NewFactory(minArgsPerProcess, pruneInterval)

	var err error
	ad, err = New(storage, plopStorage, searcher, p)
	utils.CrashOnError(errors.Wrap(err, "unable to load datastore for process indicators"))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
