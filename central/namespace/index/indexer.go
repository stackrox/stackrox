package index

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	storage "github.com/stackrox/rox/generated/storage"
	search "github.com/stackrox/rox/pkg/search"
	blevesearch "github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer is the namespace indexer.
//
//go:generate mockgen-wrapper
type Indexer interface {
	AddNamespaceMetadata(namespacemetadata *storage.NamespaceMetadata) error
	AddNamespaceMetadatas(namespacemetadatas []*storage.NamespaceMetadata) error
	Count(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNamespaceMetadata(id string) error
	DeleteNamespaceMetadatas(ids []string) error
	Search(ctx context.Context, q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
