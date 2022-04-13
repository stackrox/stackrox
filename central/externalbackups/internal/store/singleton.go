package store

import (
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	s Store
)

// Singleton returns the global external backup store
func Singleton() Store {
	once.Do(func() {
		s = New(globaldb.GetGlobalDB())
	})
	return s
}
