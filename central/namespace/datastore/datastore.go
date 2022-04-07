package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	cveSAC "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/dackbox"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	"github.com/stackrox/rox/central/idmap"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/index/mappings"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/derivedfields/counter"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/derivedfields"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sorted"
)

//go:generate mockgen-wrapper

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	GetNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error)
	AddNamespace(context.Context, *storage.NamespaceMetadata) error
	UpdateNamespace(context.Context, *storage.NamespaceMetadata) error
	RemoveNamespace(ctx context.Context, id string) error

	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store and indexer
func New(store store.Store, graphProvider graph.Provider, indexer index.Indexer, deploymentDataStore deploymentDataStore.DataStore, namespaceRanker *ranking.Ranker, idMapStorage idmap.Storage) (DataStore, error) {
	ds := &datastoreImpl{
		store:             store,
		indexer:           indexer,
		formattedSearcher: formatSearcher(indexer, graphProvider, namespaceRanker),
		deployments:       deploymentDataStore,
		namespaceRanker:   namespaceRanker,
		idMapStorage:      idMapStorage,
	}
	if err := ds.buildIndex(context.TODO()); err != nil {
		return nil, err
	}
	return ds, nil
}

var (
	namespaceSAC             = sac.ForResource(resources.Namespace)
	namespaceSACSearchHelper = namespaceSAC.MustCreateSearchHelper(mappings.OptionsMap)

	log = logging.LoggerForModule()

	defaultSortOption = &v1.QuerySortOption{
		Field:    search.Namespace.String(),
		Reversed: false,
	}
)

type datastoreImpl struct {
	store             store.Store
	indexer           index.Indexer
	formattedSearcher search.Searcher
	namespaceRanker   *ranking.Ranker

	idMapStorage idmap.Storage

	deployments deploymentDataStore.DataStore
}

func (b *datastoreImpl) buildIndex(ctx context.Context) error {
	log.Info("[STARTUP] Indexing namespaces")
	var namespaces []*storage.NamespaceMetadata
	err := b.store.Walk(ctx, func(ns *storage.NamespaceMetadata) error {
		namespaces = append(namespaces, ns)
		return nil
	})
	if err != nil {
		return err
	}

	if b.idMapStorage != nil {
		b.idMapStorage.OnNamespaceAdd(namespaces...)
	}

	if err := b.indexer.AddNamespaceMetadatas(namespaces); err != nil {
		return err
	}
	log.Info("[STARTUP] Successfully indexed namespaces")
	return nil
}

// GetNamespace returns namespace with given id.
func (b *datastoreImpl) GetNamespace(ctx context.Context, id string) (namespace *storage.NamespaceMetadata, exists bool, err error) {
	namespace, found, err := b.store.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if ok, err := namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS,
		sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())).
		Allowed(ctx); err != nil || !ok {
		return nil, false, err
	}
	b.updateNamespacePriority(namespace)
	return namespace, true, err
}

// GetNamespaces retrieves namespaces matching the request from bolt
func (b *datastoreImpl) GetNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error) {
	var allowedNamespaces []*storage.NamespaceMetadata
	err := b.store.Walk(ctx, func(namespace *storage.NamespaceMetadata) error {
		scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())}
		if ok, err := namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, scopeKeys...).
			Allowed(ctx); err != nil || !ok {
			return err
		}
		allowedNamespaces = append(allowedNamespaces, namespace)
		return nil
	})
	if err != nil {
		return nil, err
	}
	b.updateNamespacePriority(allowedNamespaces...)
	return allowedNamespaces, nil
}

// AddNamespace adds a namespace to bolt
func (b *datastoreImpl) AddNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := b.store.Upsert(ctx, namespace); err != nil {
		return err
	}
	if b.idMapStorage != nil {
		b.idMapStorage.OnNamespaceAdd(namespace)
	}
	return b.indexer.AddNamespaceMetadata(namespace)
}

// UpdateNamespace updates a namespace to bolt
func (b *datastoreImpl) UpdateNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := b.store.Upsert(ctx, namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespaceMetadata(namespace)
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
	if b.idMapStorage != nil {
		b.idMapStorage.OnNamespaceRemove(id)
	}
	// Remove ranker record here since removal is not handled in risk store as no entry present for namespace
	b.namespaceRanker.Remove(id)

	return b.indexer.DeleteNamespaceMetadata(id)
}

func (b *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return b.formattedSearcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (b *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return b.formattedSearcher.Count(ctx, q)
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
	b.updateNamespacePriority(namespaces...)
	return namespaces, err
}

func (b *datastoreImpl) updateNamespacePriority(nss ...*storage.NamespaceMetadata) {
	for _, ns := range nss {
		ns.Priority = b.namespaceRanker.GetRankForID(ns.GetId())
	}
}

// Helper functions which format our searching.
///////////////////////////////////////////////

func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher, graphProvider graph.Provider, namespaceRanker *ranking.Ranker) search.Searcher {
	filteredSearcher := namespaceSACSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, filteredSearcher, namespaceRanker)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher, namespaceRanker *ranking.Ranker) search.Searcher {
	prioritySortedSearcher := sorted.Searcher(searcher, search.NamespacePriority, namespaceRanker)

	return derivedfields.CountSortedSearcher(prioritySortedSearcher, map[string]counter.DerivedFieldCounter{
		search.DeploymentCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.NamespaceToDeploymentPath, deploymentSAC.GetSACFilter()),
		search.ImageCount.String():      counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.NamespaceToImagePath, imageSAC.GetSACFilter()),
		search.CVECount.String():        counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.NamespaceToCVEPath, cveSAC.GetSACFilter()),
	})
}
