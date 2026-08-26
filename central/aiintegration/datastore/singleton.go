package datastore

import (
	pgStore "github.com/stackrox/rox/central/aiintegration/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	instance DataStore
	once     sync.Once
)

// Singleton returns the singleton instance of the AI integration DataStore.
func Singleton() DataStore {
	once.Do(func() {
		store := pgStore.New(globaldb.GetPostgres())
		instance = New(store)
	})
	return instance
}
