package datastore

import (
	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/deployment/cache"
	"github.com/stackrox/rox/central/deployment/datastore/internal/processtagsstore"
	"github.com/stackrox/rox/central/globaldb"
	globalDackBox "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/central/globalindex"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	pbDS "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/processindicator/filter"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	var dackBox *dackbox.DackBox
	var keyFence concurrency.KeyFence
	var bleveIndex, processIndex bleve.Index
	var pool *pgxpool.Pool
	if features.PostgresDatastore.Enabled() {
		pool = globaldb.GetPostgres()
	} else {
		dackBox = globalDackBox.GetGlobalDackBox()
		keyFence = globalDackBox.GetKeyFence()
		bleveIndex = globalindex.GetGlobalIndex()
		processIndex = globalindex.GetProcessIndex()
	}
	ad = New(dackBox,
		keyFence,
		pool,
		// Process Tag store will be removed in 72.0
		processtagsstore.New(globaldb.GetGlobalDB()),
		bleveIndex,
		processIndex,
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
