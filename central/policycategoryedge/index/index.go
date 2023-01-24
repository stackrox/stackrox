package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer indexes policy category associations/edges
type Indexer interface {
	AddPolicyCategoryEdge(clustercveedge *storage.PolicyCategoryEdge) error
	AddPolicyCategoryEdges(clustercveedges []*storage.PolicyCategoryEdge) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeletePolicyCategoryEdge(id string) error
	DeletePolicyCategoryEdges(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
