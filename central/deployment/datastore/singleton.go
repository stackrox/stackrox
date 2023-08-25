package datastore

import (
	"github.com/stackrox/rox/central/deployment/cache"
	"github.com/stackrox/rox/central/globaldb"
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
	var err error
	ad, err = New(globaldb.GetPostgres(), imageDatastore.Singleton(), pbDS.Singleton(), nfDS.Singleton(), riskDS.Singleton(), cache.DeletedDeploymentCacheSingleton(), filter.Singleton(), ranking.ClusterRanker(), ranking.NamespaceRanker(), ranking.DeploymentRanker())
	if err != nil {
		log.Fatalf("could not initialize deployment datastore: %v", err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
