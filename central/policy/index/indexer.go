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

// Indexer provides indexing of Policy objects.
type Indexer interface {
	AddPolicy(policy *v1.Policy) error
	AddPolicies(policies []*v1.Policy) error
	DeletePolicy(id string) error
	SearchPolicies(request *v1.ParsedSearchRequest) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
