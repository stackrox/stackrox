package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	"github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/risk/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to RiskStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error)

	GetRisk(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) (*storage.Risk, bool, error)
	GetRiskForDeployment(ctx context.Context, deployment *storage.Deployment) (*storage.Risk, bool, error)
	UpsertRisk(ctx context.Context, risk *storage.Risk) error
	RemoveRisk(ctx context.Context, subjectID string, subjectType storage.RiskSubjectType) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(riskStore store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  riskStore,
		indexer:  indexer,
		searcher: searcher,
		subjectTypeToRanker: map[string]*ranking.Ranker{
			storage.RiskSubjectType_CLUSTER.String():         ranking.ClusterRanker(),
			storage.RiskSubjectType_NAMESPACE.String():       ranking.NamespaceRanker(),
			storage.RiskSubjectType_NODE.String():            ranking.NodeRanker(),
			storage.RiskSubjectType_NODE_COMPONENT.String():  ranking.ComponentRanker(),
			storage.RiskSubjectType_DEPLOYMENT.String():      ranking.DeploymentRanker(),
			storage.RiskSubjectType_IMAGE.String():           ranking.ImageRanker(),
			storage.RiskSubjectType_IMAGE_COMPONENT.String(): ranking.ComponentRanker(),
		},
	}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Risk)))
	if err := d.buildIndex(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	return d, nil

}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	dbstore := rocksdb.New(rocksengine)
	indexer := index.New(bleveIndex)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}
