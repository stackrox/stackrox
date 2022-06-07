package index

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Indexer provides funtionality to index node components.
type Indexer interface {
	AddNodeComponent(components *storage.NodeComponent) error
	AddNodeComponents(components []*storage.NodeComponent) error
	Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeComponent(id string) error
	DeleteNodeComponents(ids []string) error
	MarkInitialIndexingComplete() error
	NeedsInitialIndexing() (bool, error)
	Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
