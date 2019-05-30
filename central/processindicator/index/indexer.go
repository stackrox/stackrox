package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/processindicator/index/internal/index"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Policy objects.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	index.Indexer
	DeleteProcessIndicators(ids ...string) error
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(i bleve.Index) Indexer {
	return &indexerImpl{
		bleveIndex: i,
		Indexer:    index.New(i),
	}
}
