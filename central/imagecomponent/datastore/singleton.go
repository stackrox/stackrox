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
	"github.com/stackrox/stackrox/central/imagecomponent/search"
	"github.com/stackrox/stackrox/central/imagecomponent/store/dackbox"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/central/ranking"
	riskDataStore "github.com/stackrox/stackrox/central/risk/datastore"
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

	searcher := search.New(storage, globaldb.GetGlobalDackBox(),
		cveIndexer.New(globalindex.GetGlobalIndex()),
		componentCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		imageComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageIndexer.New(globalindex.GetGlobalIndex()),
		nodeComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		nodeIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex(), globalindex.GetProcessIndex()),
		clusterIndexer.New(globalindex.GetGlobalTmpIndex()))

	ad, err = New(globaldb.GetGlobalDackBox(), storage, componentIndexer.New(globalindex.GetGlobalIndex()), searcher, riskDataStore.Singleton(), ranking.ComponentRanker())
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
