package datastore

import (
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/central/imagecomponentedge/index"
	"github.com/stackrox/stackrox/central/imagecomponentedge/search"
	"github.com/stackrox/stackrox/central/imagecomponentedge/store/dackbox"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage, err := dackbox.New(globaldb.GetGlobalDackBox())
	utils.CrashOnError(err)

	searcher := search.New(storage, index.New(globalindex.GetGlobalIndex()))

	ad, err = New(globaldb.GetGlobalDackBox(), storage, index.New(globalindex.GetGlobalIndex()), searcher)
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
