package datastore

import (
	clusterIndexer "github.com/stackrox/rox/central/cluster/index"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/globaldb"
	globalDackbox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/imagecveedge/datastore/postgres"
	imageCVEEdgeIndexer "github.com/stackrox/rox/central/imagecveedge/index"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store"
	"github.com/stackrox/rox/central/imagecveedge/store/dackbox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var storage store.Store
	var indexer imageCVEEdgeIndexer.Indexer
	var searcher search.Searcher

	if features.PostgresDatastore.Enabled() {
		storage = postgres.New(globaldb.GetPostgres())
		indexer = postgres.NewIndexer(globaldb.GetPostgres())
		searcher = search.NewV2(storage, indexer)
		ad = New(nil, storage, searcher)
		return
	}

	storage = dackbox.New(globalDackbox.GetGlobalDackBox(), globalDackbox.GetKeyFence())
	indexer = imageCVEEdgeIndexer.New(globalindex.GetGlobalIndex())
	searcher = search.New(storage, cveIndexer.New(globalindex.GetGlobalIndex()),
		indexer,
		componentCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		imageComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex(), globalindex.GetProcessIndex()),
		clusterIndexer.New(globalindex.GetGlobalIndex()))

	ad = New(globalDackbox.GetGlobalDackBox(), storage, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
