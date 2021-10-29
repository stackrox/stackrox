package datastore

import (
	globaldb "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/imagecveedge/index"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := dackbox.New(globaldb.GetGlobalDackBox())

	var searcher search.Searcher
	if features.VulnRiskManagement.Enabled() {
		searcher = search.New(storage, index.New(globalindex.GetGlobalIndex()))
	}

	ad = New(globaldb.GetGlobalDackBox(), storage, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
