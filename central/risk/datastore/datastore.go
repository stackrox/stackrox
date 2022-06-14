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
	postgresStore "github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	rocksStore "github.com/stackrox/rox/central/risk/datastore/internal/store/rocksdb"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"gorm.io/gorm"
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

// GetTestPostgresDataStore provides a risk datastore hooked on rocksDB and bleve for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, _ *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) (DataStore, error) {
	postgresStore.Destroy(ctx, pool)
	storage := postgresStore.CreateTableAndNewStore(ctx, pool, gormDB)
	indexer := postgresStore.NewIndexer(pool)
	searcher := search.New(storage, indexer)
	return New(storage, indexer, searcher)
}

// GetTestRocksBleveDataStore provides a risk datastore hooked on rocksDB and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, rocksEngine *rocksdb.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	storage := rocksStore.New(rocksEngine)
	indexer := index.New(bleveIndex)
	searcher := search.New(storage, indexer)
	return New(storage, indexer, searcher)
}
