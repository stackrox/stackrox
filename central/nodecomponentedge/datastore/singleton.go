package datastore

import (
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/central/nodecomponentedge/search"
	"github.com/stackrox/stackrox/central/nodecomponentedge/store/dackbox"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := dackbox.New(globaldb.GetGlobalDackBox())
	searcher := search.New(storage, index.New(globalindex.GetGlobalIndex()))

	ad = New(globaldb.GetGlobalDackBox(), storage, index.New(globalindex.GetGlobalIndex()), searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
