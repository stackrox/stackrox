package store

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/serviceidentities/internal/store/bolt"
	"github.com/stackrox/stackrox/central/serviceidentities/internal/store/postgres"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

func initialize() {
	if features.PostgresDatastore.Enabled() {
		s = postgres.New(globaldb.GetPostgres())
	} else {
		s = bolt.New(globaldb.GetGlobalDB())
	}
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Store {
	once.Do(initialize)
	return s
}
