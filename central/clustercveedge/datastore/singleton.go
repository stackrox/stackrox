package datastore

import (
	"github.com/stackrox/stackrox/central/clustercveedge/index"
	"github.com/stackrox/stackrox/central/clustercveedge/search"
	"github.com/stackrox/stackrox/central/clustercveedge/store/dackbox"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
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
