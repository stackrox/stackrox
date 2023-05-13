package datastore

import (
	"github.com/pkg/errors"
	pgStore "github.com/stackrox/rox/central/continuousintegration/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once     sync.Once
	instance DataStore
)

// Singleton returns the singleton instance of the DataStore service.
func Singleton() DataStore {
	once.Do(func() {
		if !env.PostgresDatastoreEnabled.BooleanSetting() {
			utils.Must(errors.New("Cannot use continuous integration configs without postgres"))
		}
		store := pgStore.New(globaldb.GetPostgres())
		instance = New(store)
	})
	return instance
}
