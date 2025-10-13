package datastore

import (
	"time"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processindicator/pruner"
	pgStore "github.com/stackrox/rox/central/processindicator/store/postgres"
	plopStore "github.com/stackrox/rox/central/processlisteningonport/store/postgres"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
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
	db := globaldb.GetPostgres()
	storage := pgStore.New(db)
	plopStorage := plopStore.New(db)

	p := pruner.NewFactory(minArgsPerProcess, pruneInterval)

	ad = New(db, storage, plopStorage, p)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
