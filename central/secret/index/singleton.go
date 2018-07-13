package index

import (
	"sync"

	globalIndex "bitbucket.org/stack-rox/apollo/central/globalindex/singletons"
)

var (
	once sync.Once

	indexer Indexer
)

func initialize() {
	indexer = New(globalIndex.GetGlobalIndex())
}

// Singleton provides the instance of the Indexer interface to register.
func Singleton() Indexer {
	once.Do(initialize)
	return indexer
}
