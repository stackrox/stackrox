package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the node indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddNode(node *storage.Node) error
	AddNodes(nodes []*storage.Node) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNode(id string) error
	DeleteNodes(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
