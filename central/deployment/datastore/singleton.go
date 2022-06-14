package datastore

import (
	"github.com/stackrox/rox/central/deployment/cache"
	"github.com/stackrox/rox/central/deployment/datastore/internal/processtagsstore"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	ad = New(dackbox.GetGlobalDackBox(),
		dackbox.GetKeyFence(),
		processtagsstore.New(globaldb.GetGlobalDB()),
		globalindex.GetGlobalIndex(),
		globalindex.GetProcessIndex(),
		imageDatastore.Singleton(),
		pbDS.Singleton(),
		nfDS.Singleton(),
		riskDS.Singleton(),
		cache.DeletedDeploymentCacheSingleton(),
		filter.Singleton(),
		ranking.ClusterRanker(),
		ranking.NamespaceRanker(),
		ranking.DeploymentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
