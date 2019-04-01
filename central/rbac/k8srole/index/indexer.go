package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/storage"
)

// Indexer indexes k8s role information.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	UpsertRole(role *storage.K8SRole) error
	UpsertRoles(...*storage.K8SRole) error
	RemoveRole(id string) error
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
