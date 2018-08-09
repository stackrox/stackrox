package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/api/v1"
)

// Indexer indexes secret information.
//go:generate mockery -name=Indexer
type Indexer interface {
	SecretAndRelationship(sar *v1.SecretAndRelationship) error
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
