package store

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/serviceidentities/internal/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

func initialize() {
	s = pgStore.New(globaldb.GetPostgres())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Store {
	once.Do(initialize)
	return s
}
