package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/watchedimage/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance DataStore
	once     sync.Once
)

// Singleton returns the instance of DataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		store := pgStore.New(globaldb.GetPostgres())
		instance = New(store)
	})
	return instance
}
