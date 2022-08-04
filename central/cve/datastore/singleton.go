package datastore

import (
	clusterIndexer "github.com/stackrox/rox/central/cluster/index"
	clusterCVEEdgeIndexer "github.com/stackrox/rox/central/clustercveedge/index"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/cve/search"
	"github.com/stackrox/rox/central/cve/store/dackbox"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	globaldb "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/rox/central/imagecveedge/index"
	nodeIndexer "github.com/stackrox/rox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	globalDackbox := globaldb.GetGlobalDackBox()
	globalIndex := globalindex.GetGlobalIndex()

	storage := dackbox.New(globalDackbox, globaldb.GetKeyFence())
	searcher := search.New(storage, globalDackbox,
		cveIndexer.New(globalIndex),
		clusterCVEEdgeIndexer.New(globalIndex),
		componentCVEEdgeIndexer.New(globalIndex),
		componentIndexer.New(globalIndex),
		imageComponentEdgeIndexer.New(globalIndex),
		imageCVEEdgeIndexer.New(globalIndex),
		imageIndexer.New(globalIndex),
		nodeComponentEdgeIndexer.New(globalIndex),
		nodeIndexer.New(globalIndex),
		deploymentIndexer.New(globalIndex, globalindex.GetProcessIndex()),
		clusterIndexer.New(globalindex.GetGlobalTmpIndex()))

	var err error
	ds, err = New(globalDackbox, globaldb.GetIndexQueue(), storage, cveIndexer.New(globalIndex), searcher)
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
