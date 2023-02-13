package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/serviceidentities/internal/store/bolt"
	pgStore "github.com/stackrox/rox/central/serviceidentities/internal/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

func initialize() {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		s = pgStore.New(globaldb.GetPostgres())
	} else {
		s = bolt.New(globaldb.GetGlobalDB())
	}
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Store {
	once.Do(initialize)
	return s
}
