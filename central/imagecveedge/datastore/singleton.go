package datastore

import (
	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	"github.com/stackrox/stackrox/central/imagecveedge/search"
	"github.com/stackrox/stackrox/central/imagecveedge/store/dackbox"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := dackbox.New(globaldb.GetGlobalDackBox(), globaldb.GetKeyFence())
	var searcher = search.New(storage, cveIndexer.New(globalindex.GetGlobalIndex()),
		imageCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		imageComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex(), globalindex.GetProcessIndex()),
		clusterIndexer.New(globalindex.GetGlobalIndex()))

	ad = New(globaldb.GetGlobalDackBox(), storage, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
