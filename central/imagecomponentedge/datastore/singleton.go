package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/imagecomponentedge/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/central/imagecomponentedge/store"
	pg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error
	storage := pgStore.New(globaldb.GetPostgres())
	searcher := search.NewV2(storage)
	ad, err = New(storage, searcher)
	utils.CrashOnError(err)
}

// Constructor of the image component edge storage takes a Store object as an
// argument, but it's an internal type which could not be constructed outside.
// Such approach limits the ways how the storage could be instantiated to
// only the singleton. This method allows to avoid the limitation.
func NewStorage(db pg.DB) store.Store {
	return pgStore.New(db)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
