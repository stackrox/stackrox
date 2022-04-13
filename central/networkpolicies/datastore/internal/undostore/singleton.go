package undostore

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	undoStoreInstance     UndoStore
	undoStoreInstanceInit sync.Once
)

// Singleton returns the singleton instance of the undo store.
func Singleton() UndoStore {
	undoStoreInstanceInit.Do(func() {
		undoStoreInstance = New(globaldb.GetGlobalDB())
	})
	return undoStoreInstance
}
