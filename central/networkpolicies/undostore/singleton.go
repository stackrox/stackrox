package undostore

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
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
