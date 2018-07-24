package search

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/globalindex"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
)

var (
	once sync.Once

	searcher Searcher
)

func initialize() {
	searcher = New(store.Singleton(), globalindex.GetGlobalIndex())
}

// Singleton provides the instance of the Searcher interface to register.
func Singleton() Searcher {
	once.Do(initialize)
	return searcher
}
