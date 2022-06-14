package index

import (
	"github.com/blevesearch/bleve"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
)

//go:generate mockgen-wrapper
// Indexer is the interface for indexing active component
type Indexer interface {
	Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}

// New creates a new active component indexer
func New(index bleve.Index) Indexer {
	return &indexerImpl{index: index}
}
