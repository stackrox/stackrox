package index

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
)

// Indexer provides funtionality to index node components.
type Indexer interface {
	AddNodeComponent(components *storage.NodeComponent) error
	AddNodeComponents(components []*storage.NodeComponent) error
	Count(q *v1.Query, opts ...blevesearch.SearchOption) (int, error)
	DeleteNodeComponent(id string) error
	DeleteNodeComponents(ids []string) error
	Search(q *v1.Query, opts ...blevesearch.SearchOption) ([]search.Result, error)
}
