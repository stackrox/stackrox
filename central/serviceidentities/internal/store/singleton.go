package store

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

func initialize() {
	s = New(globaldb.GetGlobalDB())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Store {
	once.Do(initialize)
	return s
}
