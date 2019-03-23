package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/storage"
)

// Indexer indexes service account information.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	UpsertServiceAccount(*storage.ServiceAccount) error
	UpsertServiceAccounts(...*storage.ServiceAccount) error
	RemoveServiceAccount(id string) error
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
