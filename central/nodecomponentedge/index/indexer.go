package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the node-component edge indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddNodeComponentEdge(nodecomponentedge *storage.NodeComponentEdge) error
	AddNodeComponentEdges(nodecomponentedges []*storage.NodeComponentEdge) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeComponentEdge(id string) error
	DeleteNodeComponentEdges(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
