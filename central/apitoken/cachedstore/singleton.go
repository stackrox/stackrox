package cachedstore

import (
	"fmt"
	"sync"

	"github.com/stackrox/rox/central/apitoken/store"
	"github.com/stackrox/rox/central/globaldb"
)

var (
	cs CachedStore

	once sync.Once
)

func initialize() {
	var err error
	cs, err = New(store.New(globaldb.GetGlobalDB()))
	if err != nil {
		panic(fmt.Sprintf("failed to initialize APIToken store: %s", err))
	}
}

// Singleton returns the instance of CachedStore to use.
func Singleton() CachedStore {
	once.Do(initialize)
	return cs
}
