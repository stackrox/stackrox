package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/api/v1"
)

// Indexer indexes secret information.
//go:generate mockery -name=Indexer
type Indexer interface {
	UpsertSecret(sar *v1.Secret) error
	RemoveSecret(id string) error
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
