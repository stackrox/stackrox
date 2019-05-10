package store

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
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
