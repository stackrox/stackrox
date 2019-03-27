package index

import (
	"github.com/blevesearch/bleve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Deployment objects.
type Indexer interface {
	AddDeployment(deployment *storage.Deployment) error
	AddDeployments(deployments []*storage.Deployment) error
	DeleteDeployment(id string) error
	Search(q *v1.Query) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
