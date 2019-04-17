package index

import (
	"github.com/blevesearch/bleve"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Indexer indexes k8s role binding information.
//go:generate mockgen-wrapper Indexer
type Indexer interface {
	UpsertRoleBinding(role *storage.K8SRoleBinding) error
	UpsertRoleBindings(...*storage.K8SRoleBinding) error
	RemoveRoleBinding(id string) error
	Search(q *v1.Query) ([]search.Result, error)
}

// New provides a new Indexer using the given bleve index underneath.
func New(index bleve.Index) Indexer {
	return &indexerImpl{
		index: index,
	}
}
