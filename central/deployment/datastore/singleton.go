package datastore

import (
	"github.com/stackrox/stackrox/central/deployment/cache"
	"github.com/stackrox/stackrox/central/deployment/datastore/internal/processtagsstore"
	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/globaldb/dackbox"
	"github.com/stackrox/stackrox/central/globalindex"
	imageDatastore "github.com/stackrox/stackrox/central/image/datastore"
	nfDS "github.com/stackrox/stackrox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/stackrox/central/processbaseline/datastore"
	"github.com/stackrox/stackrox/central/processindicator/filter"
	"github.com/stackrox/stackrox/central/ranking"
	riskDS "github.com/stackrox/stackrox/central/risk/datastore"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/sync"
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
