package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/index/mappings"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

//go:generate mockgen-wrapper DataStore

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	GetNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error)
	AddNamespace(context.Context, *storage.NamespaceMetadata) error
	UpdateNamespace(context.Context, *storage.NamespaceMetadata) error
	RemoveNamespace(ctx context.Context, id string) error

	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	SearchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store and indexer
func New(store store.Store, indexer index.Indexer) (DataStore, error) {
	ds := &datastoreImpl{
		store:      store,
		indexer:    indexer,
		keyedMutex: concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	return ds, nil
}

var (
	namespaceSAC             = sac.ForResource(resources.Namespace)
	namespaceSACSearchHelper = namespaceSAC.MustCreateSearchHelper(mappings.OptionsMap, true)
)

type datastoreImpl struct {
	store   store.Store
	indexer index.Indexer

	keyedMutex *concurrency.KeyedMutex
}

func (b *datastoreImpl) buildIndex() error {
	namespaces, err := b.store.GetNamespaces()
	if err != nil {
		return err
	}
	return b.indexer.AddNamespaces(namespaces)
}

// GetNamespace returns namespace with given id.
func (b *datastoreImpl) GetNamespace(ctx context.Context, id string) (namespace *storage.NamespaceMetadata, exists bool, err error) {
	namespace, found, err := b.store.GetNamespace(id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS,
		sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())).
		Allowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	return namespace, true, err
}

// GetNamespaces retrieves namespaces matching the request from bolt
func (b *datastoreImpl) GetNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error) {
	namespaces, err := b.store.GetNamespaces()
	if err != nil {
		return nil, err
	}

	allowedNamespaces := make([]*storage.NamespaceMetadata, len(namespaces))
	for _, namespace := range namespaces {
		scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())}
		if ok, err := namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, scopeKeys...).
			Allowed(ctx); err != nil || !ok {
			continue
		}
		allowedNamespaces = append(allowedNamespaces, namespace)
	}

	return allowedNamespaces, nil
}

// AddNamespace adds a namespace to bolt
func (b *datastoreImpl) AddNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	b.keyedMutex.Lock(namespace.GetId())
	defer b.keyedMutex.Unlock(namespace.GetId())
	if err := b.store.AddNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespace(namespace)
}

// UpdateNamespace updates a namespace to bolt
func (b *datastoreImpl) UpdateNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	b.keyedMutex.Lock(namespace.GetId())
	defer b.keyedMutex.Unlock(namespace.GetId())
	if err := b.store.UpdateNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespace(namespace)
}

// RemoveNamespace removes a namespace.
func (b *datastoreImpl) RemoveNamespace(ctx context.Context, id string) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	b.keyedMutex.Lock(id)
	defer b.keyedMutex.Unlock(id)
	if err := b.store.RemoveNamespace(id); err != nil {
		return err
	}
	return b.indexer.DeleteNamespace(id)
}

func (b *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return namespaceSACSearchHelper.Apply(b.indexer.Search)(ctx, q)
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

func (b *datastoreImpl) searchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, []search.Result, error) {
	results, err := b.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	if len(results) == 0 {
		return nil, nil, nil
	}
	nsSlice := make([]*storage.NamespaceMetadata, 0, len(results))
	resultSlice := make([]search.Result, 0, len(results))
	for _, res := range results {
		ns, exists, err := b.GetNamespace(ctx, res.ID)
		if err != nil {
			return nil, resultSlice, errors.Wrapf(err, "retrieving namespace %q", res.ID)
		}
		if !exists {
			// This could be due to a race where it's deleted in the time between
			// the search and the query to Bolt.
			continue
		}
		nsSlice = append(nsSlice, ns)
		resultSlice = append(resultSlice, res)
	}
	return nsSlice, resultSlice, nil
}

func (b *datastoreImpl) SearchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, error) {
	namespaces, _, err := b.searchNamespaces(ctx, q)
	return namespaces, err
}
