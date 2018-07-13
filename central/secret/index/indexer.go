package index

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
)

// Indexer indexes secret information.
type Indexer interface {
	SecretAndRelationship(sar *v1.SecretAndRelationship) error
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
