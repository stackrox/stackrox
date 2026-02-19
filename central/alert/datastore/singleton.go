package datastore

import (
	pgStore "github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	platformmatcher "github.com/stackrox/rox/central/platform/matcher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance DataStore
)

func initialize() {
	db := globaldb.GetPostgres()
	storage := pgStore.New(db)
	soleInstance = New(db, storage, platformmatcher.Singleton())
}

// Singleton returns the sole instance of the DataStore service.
func Singleton() DataStore {
	once.Do(initialize)
	return soleInstance
}
