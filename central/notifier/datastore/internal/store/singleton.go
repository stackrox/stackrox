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

// Singleton provides the global store instance. This should only be used by the DataStore to enforce access control.
func Singleton() Store {
	once.Do(initialize)
	return as
}
