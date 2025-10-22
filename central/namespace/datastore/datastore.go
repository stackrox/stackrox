package datastore

import (
	"context"
	"fmt"
	"iter"
	"slices"

	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore/internal/store"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sorted"
)

//go:generate mockgen-wrapper

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(ctx context.Context, id string) (storage.ImmutableNamespaceMetadata, bool, error)
	GetAllNamespaces(ctx context.Context) ([]storage.ImmutableNamespaceMetadata, error)
	GetNamespacesForSAC(ctx context.Context) ([]storage.ImmutableNamespaceMetadata, error)
	GetManyNamespaces(ctx context.Context, id []string) ([]storage.ImmutableNamespaceMetadata, error)

	AddNamespace(context.Context, storage.ImmutableNamespaceMetadata) error
	UpdateNamespace(context.Context, storage.ImmutableNamespaceMetadata) error
	RemoveNamespace(ctx context.Context, id string) error

	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchNamespaces(ctx context.Context, q *v1.Query) ([]storage.ImmutableNamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store.
func New(nsStore store.Store, deploymentDataStore deploymentDataStore.DataStore, namespaceRanker *ranking.Ranker) DataStore {
	return &datastoreImpl{
		store:           nsStore,
		deployments:     deploymentDataStore,
		namespaceRanker: namespaceRanker,
	}
}

var (
	namespaceSAC = sac.ForResource(resources.Namespace)
)

type datastoreImpl struct {
	store           store.Store
	namespaceRanker *ranking.Ranker

	deployments deploymentDataStore.DataStore
}

// GetNamespace returns namespace with given id.
func (b *datastoreImpl) GetNamespace(ctx context.Context, id string) (namespace storage.ImmutableNamespaceMetadata, exists bool, err error) {
	namespace, found, err := b.store.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if !namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS,
		sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())).
		IsAllowed() {
		return nil, false, nil
	}
	return b.updateNamespacePriority(namespace.CloneVT()), true, err
}

// GetAllNamespaces retrieves namespaces matching the request
func (b *datastoreImpl) GetAllNamespaces(ctx context.Context) ([]storage.ImmutableNamespaceMetadata, error) {
	var allowedNamespaces []storage.ImmutableNamespaceMetadata
	walkFn := func() error {
		allowedNamespaces = allowedNamespaces[:0]
		return b.store.Walk(ctx, func(namespace *storage.NamespaceMetadata) error {
			scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())}
			if !namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, scopeKeys...).
				IsAllowed() {
				return nil
			}
			allowedNamespaces = append(allowedNamespaces, namespace.CloneVT())
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}
	return b.updateNamespacesPriority(allowedNamespaces), nil
}

// GetNamespacesForSAC retrieves namespaces matching the request
func (b *datastoreImpl) GetNamespacesForSAC(ctx context.Context) ([]storage.ImmutableNamespaceMetadata, error) {
	ok, err := namespaceSAC.ReadAllowed(ctx)
	if err != nil {
		return nil, err
	} else if !ok {
		return b.SearchNamespaces(ctx, search.EmptyQuery())
	}
	var allowedNamespaces []storage.ImmutableNamespaceMetadata
	walkFn := func() error {
		allowedNamespaces = allowedNamespaces[:0]
		return b.store.Walk(ctx, func(namespace *storage.NamespaceMetadata) error {
			scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())}
			if !namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, scopeKeys...).
				IsAllowed() {
				return nil
			}
			allowedNamespaces = append(allowedNamespaces, namespace)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}
	return allowedNamespaces, nil
}

func (b *datastoreImpl) GetManyNamespaces(ctx context.Context, ids []string) ([]storage.ImmutableNamespaceMetadata, error) {
	var err error
	if ok, err := namespaceSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		query := search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery()
		return b.SearchNamespaces(ctx, query)
	}
	namespaces, _, err := b.store.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return b.updateNamespacesPriority(slices.Collect(values(namespaces...))), nil
}

func values(nss ...*storage.NamespaceMetadata) iter.Seq[storage.ImmutableNamespaceMetadata] {
	return func(yield func(metadata storage.ImmutableNamespaceMetadata) bool) {
		for _, v := range nss {
			if !yield(v) {
				return
			}
		}
	}
}

// AddNamespace adds a namespace.
func (b *datastoreImpl) AddNamespace(ctx context.Context, namespace storage.ImmutableNamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.store.Upsert(ctx, namespace.CloneVT())
}

// UpdateNamespace updates a namespace to the database
func (b *datastoreImpl) UpdateNamespace(ctx context.Context, namespace storage.ImmutableNamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.store.Upsert(ctx, namespace.CloneVT())
}

// RemoveNamespace removes a namespace.
func (b *datastoreImpl) RemoveNamespace(ctx context.Context, id string) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := b.store.Delete(ctx, id); err != nil {
		return err
	}
	// Remove ranker record here since removal is not handled in risk store as no entry present for namespace
	b.namespaceRanker.Remove(id)

	return nil
}

func (b *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	// Need to check if we are sorting by priority.
	validPriorityQuery, err := sorted.IsValidPriorityQuery(q, search.NamespacePriority)
	if err != nil {
		return nil, err
	}
	if validPriorityQuery {
		priorityQuery, reversed, err := sorted.RemovePrioritySortFromQuery(q, search.NamespacePriority)
		if err != nil {
			return nil, err
		}
		results, err := b.store.Search(ctx, priorityQuery)
		if err != nil {
			return nil, err
		}

		sortedResults := sorted.SortResults(results, reversed, b.namespaceRanker)
		return paginated.PageResults(sortedResults, q)
	}

	return b.store.Search(ctx, q)
}

// Count returns the number of search results from the query
func (b *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return b.store.Count(ctx, q)
}

func (b *datastoreImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	namespaces, results, err := b.searchNamespaces(ctx, q)
	if err != nil {
		return nil, err
	}

	searchResults := make([]*v1.SearchResult, 0, len(namespaces))
	for i, r := range results {
		namespace := namespaces[i]
		searchResults = append(searchResults, &v1.SearchResult{
			Id:             r.ID,
			Name:           namespace.GetName(),
			Category:       v1.SearchCategory_NAMESPACES,
			Score:          r.Score,
			FieldToMatches: search.GetProtoMatchesMap(r.Matches),
			Location:       fmt.Sprintf("%s/%s", namespace.GetClusterName(), namespace.GetName()),
		})
	}
	return searchResults, nil
}

func (b *datastoreImpl) searchNamespaces(ctx context.Context, q *v1.Query) ([]storage.ImmutableNamespaceMetadata, []search.Result, error) {
	// TODO(ROX-29943): remove unnecessary calls to database
	results, err := b.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	if len(results) == 0 {
		return nil, nil, nil
	}
	nsSlice := make([]storage.ImmutableNamespaceMetadata, 0, len(results))
	resultSlice := make([]search.Result, 0, len(results))
	for _, res := range results {
		ns, exists, err := b.GetNamespace(ctx, res.ID)
		if err != nil {
			return nil, resultSlice, errors.Wrapf(err, "retrieving namespace %q", res.ID)
		}
		if !exists {
			// This could be due to a race where it's deleted in the time between
			// the search and the query.
			continue
		}
		nsSlice = append(nsSlice, ns)
		resultSlice = append(resultSlice, res)
	}
	return nsSlice, resultSlice, nil
}

func (b *datastoreImpl) SearchNamespaces(ctx context.Context, q *v1.Query) ([]storage.ImmutableNamespaceMetadata, error) {
	namespaces, _, err := b.searchNamespaces(ctx, q)
	return b.updateNamespacesPriority(namespaces), err
}

func (b *datastoreImpl) updateNamespacesPriority(namespaces []storage.ImmutableNamespaceMetadata) []storage.ImmutableNamespaceMetadata {
	results := make([]storage.ImmutableNamespaceMetadata, 0, len(namespaces))
	for _, ns := range namespaces {
		results = append(results, b.updateNamespacePriority(ns.CloneVT()))
	}
	return results
}

func (b *datastoreImpl) updateNamespacePriority(ns *storage.NamespaceMetadata) storage.ImmutableNamespaceMetadata {
	ns.Priority = b.namespaceRanker.GetRankForID(ns.GetId())
	return ns
}
