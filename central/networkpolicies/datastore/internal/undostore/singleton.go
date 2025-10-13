package undostore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/networkpolicies/datastore/internal/undostore/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	undoStoreInstance     UndoStore
	undoStoreInstanceInit sync.Once
)

// Singleton returns the singleton instance of the undo store.
func Singleton() UndoStore {
	undoStoreInstanceInit.Do(func() {
		undoStoreInstance = pgStore.New(globaldb.GetPostgres())
	})
	return undoStoreInstance
}
