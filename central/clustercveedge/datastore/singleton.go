package datastore

import (
	"github.com/stackrox/rox/central/clustercveedge/index"
	"github.com/stackrox/rox/central/clustercveedge/search"
	"github.com/stackrox/rox/central/clustercveedge/store/dackbox"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	globaldb "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage, err := dackbox.New(globaldb.GetGlobalDackBox(), globaldb.GetKeyFence())
	utils.CrashOnError(err)

	searcher := search.New(storage, index.New(globalindex.GetGlobalIndex()), cveIndexer.New(globalindex.GetGlobalIndex()), globaldb.GetGlobalDackBox())

	ad, err = New(globaldb.GetGlobalDackBox(), storage, index.New(globalindex.GetGlobalIndex()), searcher)
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
