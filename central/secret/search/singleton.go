package search

import (
	"sync"

	globalIndex "bitbucket.org/stack-rox/apollo/central/globalindex/singletons"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
)

var (
	once sync.Once

	searcher Searcher
)

func initialize() {
	searcher = New(store.Singleton(), globalIndex.GetGlobalIndex())
}

// Singleton provides the instance of the Searcher interface to register.
func Singleton() Searcher {
	once.Do(initialize)
	return searcher
}
