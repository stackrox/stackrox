package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Alert objects.
//go:generate mockery -name=Indexer
type Indexer interface {
	AddAlert(alert *v1.Alert) error
	AddAlerts(alerts []*v1.Alert) error
	DeleteAlert(id string) error
	SearchAlerts(q *v1.Query) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
