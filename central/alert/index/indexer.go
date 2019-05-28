package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/index/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Alert objects.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	AddListAlert(alert *storage.ListAlert) error
	AddListAlerts(alerts []*storage.ListAlert) error
	DeleteListAlert(id string) error
	Search(q *v1.Query) ([]search.Result, error)
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(i bleve.Index) Indexer {
	return &indexerImpl{
		Indexer: index.New(i),
	}
}
