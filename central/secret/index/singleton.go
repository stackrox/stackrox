package index

import (
	"sync"

	"github.com/stackrox/rox/central/globalindex"
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
