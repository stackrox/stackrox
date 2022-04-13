package store

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Store
)

func initialize() {
	as = New(globaldb.GetGlobalDB())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Store {
	once.Do(initialize)
	return as
}
