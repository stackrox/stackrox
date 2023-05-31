package health

import (
	pgStore "github.com/stackrox/rox/central/declarativeconfig/health/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	ds = New(pgStore.New(globaldb.GetPostgres()))
}

// Singleton returns datastore instance.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
