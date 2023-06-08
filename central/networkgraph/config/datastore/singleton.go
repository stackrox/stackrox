package datastore

import (
	"github.com/stackrox/rox/central/globaldb"

	pgStore "github.com/stackrox/rox/central/networkgraph/config/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton provides the instance of DataStore to use.
func Singleton() DataStore {
	once.Do(func() {
		instance = New(pgStore.New(globaldb.GetPostgres()))
	})
	return instance
}
