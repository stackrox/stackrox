package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to RiskStorage.
//
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

// New returns a new instance of DataStore using the input store, and searcher.
func New(riskStore store.Store, searcher search.Searcher) DataStore {
	d := &datastoreImpl{
		storage:  riskStore,
		searcher: searcher,
		entityTypeToRanker: map[string]*ranking.Ranker{
			storage.RiskSubjectType_CLUSTER.String():         ranking.ClusterRanker(),
			storage.RiskSubjectType_NAMESPACE.String():       ranking.NamespaceRanker(),
			storage.RiskSubjectType_NODE.String():            ranking.NodeRanker(),
			storage.RiskSubjectType_NODE_COMPONENT.String():  ranking.NodeComponentRanker(),
			storage.RiskSubjectType_DEPLOYMENT.String():      ranking.DeploymentRanker(),
			storage.RiskSubjectType_IMAGE.String():           ranking.ImageRanker(),
			storage.RiskSubjectType_IMAGE_COMPONENT.String(): ranking.ComponentRanker(),
		},
	}
	return d
}

// GetTestPostgresDataStore provides a datastore connected to pgStore for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, searcher)
}
