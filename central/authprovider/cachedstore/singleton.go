package cachedstore

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/authprovider/store"
	globaldb "bitbucket.org/stack-rox/apollo/central/globaldb/singletons"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
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
