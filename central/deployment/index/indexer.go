package index

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/blevesearch/bleve"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Deployment objects.
type Indexer interface {
	AddDeployment(alert *v1.Deployment) error
	DeleteDeployment(id string) error
	SearchDeployments(request *v1.ParsedSearchRequest) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
