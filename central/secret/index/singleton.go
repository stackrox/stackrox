package index

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/globalindex"
)

var (
	once sync.Once

	indexer Indexer
)

func initialize() {
	indexer = New(globalindex.GetGlobalIndex())
}

// Singleton provides the instance of the Indexer interface to register.
func Singleton() Indexer {
	once.Do(initialize)
	return indexer
}
