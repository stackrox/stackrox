package store

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
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
