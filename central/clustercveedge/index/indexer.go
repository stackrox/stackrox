package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the cluster-cve edge indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddClusterCVEEdge(clustercveedge *storage.ClusterCVEEdge) error
	AddClusterCVEEdges(clustercveedges []*storage.ClusterCVEEdge) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteClusterCVEEdge(id string) error
	DeleteClusterCVEEdges(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
