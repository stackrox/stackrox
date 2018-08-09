package search

import (
	"sync"

	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/secret/store"
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
