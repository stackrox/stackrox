package search

import (
	"context"

	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/index"
	"github.com/stackrox/rox/central/rbac/k8srolebinding/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing k8s role bindings.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchRoleBindings(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawRoleBindings(ctx context.Context, query *v1.Query) ([]*storage.K8SRoleBinding, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, index index.Indexer) Searcher {
	return &searcherImpl{
		storage: storage,
		index:   index,
	}
}
