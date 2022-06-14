package undostore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/networkpolicies/datastore/internal/undostore/bolt"
	"github.com/stackrox/stackrox/central/networkpolicies/datastore/internal/undostore/postgres"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	undoStoreInstance     UndoStore
	undoStoreInstanceInit sync.Once
)

// Singleton returns the singleton instance of the undo store.
func Singleton() UndoStore {
	undoStoreInstanceInit.Do(func() {
		if features.PostgresDatastore.Enabled() {
			undoStoreInstance = postgres.New(globaldb.GetPostgres())
		} else {
			undoStoreInstance = bolt.New(globaldb.GetGlobalDB())
		}
	})
	return undoStoreInstance
}
