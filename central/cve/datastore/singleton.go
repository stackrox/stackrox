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
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	if !features.Dackbox.Enabled() {
		ds = nil
		return
	}
	storage, err := dackbox.New(globaldb.GetGlobalDackBox(), globaldb.GetKeyFence())
	utils.Must(err)

	searcher := search.New(storage, globaldb.GetGlobalDackBox(),
		cveIndexer.New(globalindex.GetGlobalIndex()),
		clusterCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentCVEEdgeIndexer.New(globalindex.GetGlobalIndex()),
		componentIndexer.New(globalindex.GetGlobalIndex()),
		imageComponentEdgeIndexer.New(globalindex.GetGlobalIndex()),
		imageIndexer.New(globalindex.GetGlobalIndex()),
		deploymentIndexer.New(globalindex.GetGlobalIndex()),
		clusterIndexer.New(globalindex.GetGlobalTmpIndex()))

	ds, err = New(globaldb.GetGlobalDackBox(), storage, cveIndexer.New(globalindex.GetGlobalIndex()), searcher)
	utils.Must(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
