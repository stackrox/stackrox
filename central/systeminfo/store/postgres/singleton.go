package postgres

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	store Store
)

// Singleton provides the singleton instance of the system info store interface.
func Singleton() Store {
	once.Do(func() {
		store = New(globaldb.GetPostgres())
	})
	return store
}
