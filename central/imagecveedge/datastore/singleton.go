package datastore

import (
	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	"github.com/stackrox/stackrox/central/globaldb"
	globalDackbox "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	"github.com/stackrox/stackrox/central/imagecveedge/datastore/internal/postgres"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	"github.com/stackrox/stackrox/central/imagecveedge/search"
	"github.com/stackrox/stackrox/central/imagecveedge/store"
	"github.com/stackrox/stackrox/central/imagecveedge/store/dackbox"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sync"
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
