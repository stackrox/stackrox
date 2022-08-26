package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/cve/cluster/datastore/index"
	"github.com/stackrox/rox/central/cve/cluster/datastore/search"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store/postgres"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/converter/v2"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to cluster CVE storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ClusterCVE, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ClusterCVE, error)

	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error
	Unsuppress(ctx context.Context, cves ...string) error

	// UpsertInternal and DeleteInternal provide functionality to add and remove k8s, openshift and istio vulnerabilities.
	// These functions are used only by cve fetcher to periodically update cluster vulns, and should not be exposed to the service layer.

	UpsertClusterCVEsInternal(ctx context.Context, cveType storage.CVE_CVEType, cveParts ...converter.ClusterCVEParts) error
	DeleteClusterCVEsInternal(ctx context.Context, clusterID string) error
}

// New returns a new instance of a DataStore.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,

		cveSuppressionCache: make(common.CVESuppressionCache),
	}
	if err := ds.buildSuppressedCache(); err != nil {
		return nil, err
	}
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	dbstore := postgres.NewFullStore(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher)
}
