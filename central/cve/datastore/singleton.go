package datastore

import (
	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	clusterCVEEdgeIndexer "github.com/stackrox/stackrox/central/clustercveedge/index"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	"github.com/stackrox/stackrox/central/cve/search"
	"github.com/stackrox/stackrox/central/cve/store/dackbox"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	globaldb "github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := dackbox.New(globaldb.GetGlobalDackBox(), globaldb.GetKeyFence())

	searcher := search.New(storage, globaldb.GetGlobalDackBox(),
		cveIndexer.New(globalindex.GetGlobalIndex()),
		clusterCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		imageComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageIndexer.New(globalindex.GetGlobalIndex()),
		nodeComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		nodeIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex(), globalindex.GetProcessIndex()),
		clusterIndexer.New(globalindex.GetGlobalTmpIndex()))

	var err error
	ds, err = New(globaldb.GetGlobalDackBox(), globaldb.GetIndexQueue(), storage, cveIndexer.New(globalindex.GetGlobalIndex()), searcher)
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
