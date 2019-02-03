package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Namespace objects.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	AddNamespace(namespace *storage.Namespace) error
	AddNamespaces(namespaces []*storage.Namespace) error
	DeleteNamespace(id string) error
	Search(q *v1.Query) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
