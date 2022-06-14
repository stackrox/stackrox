package index

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	storage "github.com/stackrox/stackrox/generated/storage"
	search "github.com/stackrox/stackrox/pkg/search"
	blevesearch "github.com/stackrox/stackrox/pkg/search/blevesearch"
)

// Indexer provides indexing functionality for storage.NodeComponentCVEEdge objects.
type Indexer interface {
	AddNodeComponentCVEEdge(componentcveedge *storage.NodeComponentCVEEdge) error
	AddNodeComponentCVEEdges(componentcveedges []*storage.NodeComponentCVEEdge) error
	Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeComponentCVEEdge(id string) error
	DeleteNodeComponentCVEEdges(ids []string) error
	Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
