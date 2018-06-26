package singletons

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/globalindex"
	"github.com/blevesearch/bleve"
)

var (
	once sync.Once

	globalIndex bleve.Index
)

func initialize() {
	var err error
	globalIndex, err = globalindex.InitializeIndices("/tmp/moss.bleve")
	if err != nil {
		panic(err)
	}
}

// GetGlobalIndex provides the global bleve index to use for indexing.
func GetGlobalIndex() bleve.Index {
	once.Do(initialize)
	return globalIndex
}
