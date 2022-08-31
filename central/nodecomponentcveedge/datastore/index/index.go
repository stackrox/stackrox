package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer provides indexing functionality for storage.NodeComponentCVEEdge objects.
type Indexer interface {
	AddNodeComponentCVEEdge(componentcveedge *storage.NodeComponentCVEEdge) error
	AddNodeComponentCVEEdges(componentcveedges []*storage.NodeComponentCVEEdge) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeComponentCVEEdge(id string) error
	DeleteNodeComponentCVEEdges(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
