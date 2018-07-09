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

// Indexer provides indexing of Alert objects.
type Indexer interface {
	AddImage(image *v1.Image) error
	AddImages(imageList []*v1.Image) error
	DeleteImage(id string) error
	SearchImages(request *v1.ParsedSearchRequest) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
