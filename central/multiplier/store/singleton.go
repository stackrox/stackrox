package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
)

var (
	once sync.Once

	store Store
)

func initialize() {
	store = New(globaldb.GetGlobalDB())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Store {
	once.Do(initialize)
	return store
}
