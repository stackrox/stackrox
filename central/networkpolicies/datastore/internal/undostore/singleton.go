package undostore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/bolt"
	pgStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	undoStoreInstance     UndoStore
	undoStoreInstanceInit sync.Once
)

// Singleton returns the singleton instance of the undo store.
func Singleton() UndoStore {
	undoStoreInstanceInit.Do(func() {
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			undoStoreInstance = pgStore.New(globaldb.GetPostgres())
		} else {
			undoStoreInstance = bolt.New(globaldb.GetGlobalDB())
		}
	})
	return undoStoreInstance
}
