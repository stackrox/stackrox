package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/index/internal/index"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Indexer provides indexing of Alert objects.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	index.Indexer
}

// New returns a new instance of Indexer using the bleve Index provided.
func New(i bleve.Index) Indexer {
	return &indexerImpl{
		Indexer: index.New(i),
	}
}
