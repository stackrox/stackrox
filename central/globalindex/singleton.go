package globalindex

import (
	"sync"

	"github.com/blevesearch/bleve"
)

var (
	once sync.Once

	gi bleve.Index
)

func initialize() {
	var err error
	gi, err = InitializeIndices("/tmp/search/scorch.bleve")
	if err != nil {
		panic(err)
	}
}

// GetGlobalIndex provides the global bleve index to use for indexing.
func GetGlobalIndex() bleve.Index {
	once.Do(initialize)
	return gi
}
