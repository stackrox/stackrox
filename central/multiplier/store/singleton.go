package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
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
