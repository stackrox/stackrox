package store

import (
	"github.com/stackrox/rox/central/externalbackups/internal/store/bolt"
	pgStore "github.com/stackrox/rox/central/externalbackups/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

// Singleton returns the global external backup store
func Singleton() Store {
	once.Do(func() {
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			s = pgStore.New(globaldb.GetPostgres())
		} else {
			s = bolt.New(globaldb.GetGlobalDB())
		}
	})
	return s
}
