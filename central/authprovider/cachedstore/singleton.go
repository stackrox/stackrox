package cachedstore

import (
	"sync"

	"github.com/stackrox/rox/central/authprovider/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	logger = logging.LoggerForModule()

	once sync.Once

	storage store.Store

	cs CachedStore
)

func initialize() {
	storage = store.New(globaldb.GetGlobalDB())

	cs = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() CachedStore {
	once.Do(initialize)
	return cs
}
