package datastore

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/central/node/datastore/search"
	"github.com/stackrox/rox/central/node/datastore/store"
	pgStore "github.com/stackrox/rox/central/node/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to NodeStorage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchNodes(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error)

	CountNodes(ctx context.Context) (int, error)
	GetNode(ctx context.Context, id string) (*storage.Node, bool, error)
	GetNodesBatch(ctx context.Context, ids []string) ([]*storage.Node, error)
	GetManyNodeMetadata(ctx context.Context, ids []string) ([]*storage.Node, error)

	UpsertNode(ctx context.Context, node *storage.Node) error

	DeleteNodes(ctx context.Context, ids ...string) error
	DeleteAllNodesForCluster(ctx context.Context, clusterID string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// NewWithPostgres returns a new instance of DataStore using the input store, and searcher.
func NewWithPostgres(storage store.Store, searcher search.Searcher, risks riskDS.DataStore, nodeRanker *ranking.Ranker, nodeComponentRanker *ranking.Ranker) DataStore {
	ds := newDatastoreImpl(storage, searcher, risks, nodeRanker, nodeComponentRanker)
	ds.initializeRankers()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool, false, concurrency.NewKeyFence())
	indexer := pgStore.NewIndexer(pool)
	searcher := search.NewV2(dbstore, indexer)
	riskStore := riskDS.GetTestPostgresDataStore(t, pool)
	nodeRanker := ranking.NodeRanker()
	nodeComponentRanker := ranking.NodeComponentRanker()
	return NewWithPostgres(dbstore, searcher, riskStore, nodeRanker, nodeComponentRanker)
}

// NodeString returns a human-readable string representation of a node.
func NodeString(node *storage.Node) string {
	return fmt.Sprintf("%s/%s (id: %s)", node.GetClusterName(), node.GetName(), node.GetId())
}
