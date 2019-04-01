package index

import (
	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/generated/storage"
)

// Indexer indexes k8s role binding information.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	UpsertRoleBinding(role *storage.K8SRoleBinding) error
	UpsertRoleBindings(...*storage.K8SRoleBinding) error
	RemoveRoleBinding(id string) error
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
