package store

import (
	pgStore "github.com/stackrox/rox/central/externalbackups/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

// Singleton returns the global external backup store
func Singleton() Store {
	once.Do(func() {
		s = pgStore.New(globaldb.GetPostgres())
	})
	return s
}
