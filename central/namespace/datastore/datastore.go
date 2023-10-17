package datastore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/store"
	pgStore "github.com/stackrox/rox/central/namespace/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cache/objectarraycache"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sorted"
)

//go:generate mockgen-wrapper

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	GetAllNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error)
	GetNamespacesForSAC(ctx context.Context) ([]effectiveaccessscope.NamespaceForSAC, error)
	GetManyNamespaces(ctx context.Context, id []string) ([]*storage.NamespaceMetadata, error)

	AddNamespace(context.Context, *storage.NamespaceMetadata) error
	UpdateNamespace(context.Context, *storage.NamespaceMetadata) error
	RemoveNamespace(ctx context.Context, id string) error

	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store and indexer
func New(nsStore store.Store, indexer index.Indexer, deploymentDataStore deploymentDataStore.DataStore, namespaceRanker *ranking.Ranker) DataStore {
	ds := &datastoreImpl{
		store:             nsStore,
		deployments:       deploymentDataStore,
		namespaceRanker:   namespaceRanker,
		formattedSearcher: formatSearcherV2(indexer, namespaceRanker),
	}
	ds.objectCacheForSAC = objectarraycache.NewObjectArrayCache(cacheRefreshPeriod, ds.getNamespacesForSAC)
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	deploymentStore, err := deploymentDataStore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	namespaceRanker := ranking.NamespaceRanker()
	return New(dbstore, indexer, deploymentStore, namespaceRanker), nil
}

var (
	namespaceSAC = sac.ForResource(resources.Namespace)

	defaultSortOption = &v1.QuerySortOption{
		Field:    search.Namespace.String(),
		Reversed: false,
	}
)

const (
	cacheRefreshPeriod = 5 * time.Second
)

type datastoreImpl struct {
	store             store.Store
	indexer           index.Indexer
	formattedSearcher search.Searcher
	namespaceRanker   *ranking.Ranker

	deployments deploymentDataStore.DataStore

	objectCacheForSAC *objectarraycache.ObjectArrayCache[effectiveaccessscope.NamespaceForSAC]
}

// GetNamespace returns namespace with given id.
func (b *datastoreImpl) GetNamespace(ctx context.Context, id string) (namespace *storage.NamespaceMetadata, exists bool, err error) {
	namespace, found, err := b.store.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}

	if !namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS,
		sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())).
		IsAllowed() {
		return nil, false, nil
	}
	b.updateNamespacePriority(namespace)
	return namespace, true, err
}

// GetAllNamespaces retrieves namespaces matching the request
func (b *datastoreImpl) GetAllNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error) {
	var allowedNamespaces []*storage.NamespaceMetadata
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
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	b.updateNamespacePriority(allowedNamespaces...)
	return allowedNamespaces, nil
}

// GetNamespacesForSAC retrieves the namespaces for SAC check purposes
func (b *datastoreImpl) GetNamespacesForSAC(ctx context.Context) ([]effectiveaccessscope.NamespaceForSAC, error) {
	return b.objectCacheForSAC.GetObjects(ctx)
}

func (b *datastoreImpl) getNamespacesForSAC(ctx context.Context) ([]effectiveaccessscope.NamespaceForSAC, error) {
	namespaces := make([]effectiveaccessscope.NamespaceForSAC, 0)
	err := b.store.Walk(ctx, func(namespace *storage.NamespaceMetadata) error {
		namespaces = append(namespaces, storageNamespaceToNamespaceForSAC(namespace))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

func (b *datastoreImpl) GetManyNamespaces(ctx context.Context, ids []string) ([]*storage.NamespaceMetadata, error) {
	var namespaces []*storage.NamespaceMetadata
	var err error
	if ok, err := namespaceSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		query := search.NewQueryBuilder().AddDocIDs(ids...).ProtoQuery()
		return b.SearchNamespaces(ctx, query)
	}
	namespaces, _, err = b.store.GetMany(ctx, ids)
	b.updateNamespacePriority(namespaces...)
	if err != nil {
		return nil, err
	}
	return namespaces, nil
}

// AddNamespace adds a namespace.
func (b *datastoreImpl) AddNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.store.Upsert(ctx, namespace)
}

// UpdateNamespace updates a namespace to bolt
func (b *datastoreImpl) UpdateNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return b.store.Upsert(ctx, namespace)
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
			// the search and the query.
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

func formatSearcherV2(searcher search.Searcher, namespaceRanker *ranking.Ranker) search.Searcher {
	scopedSearcher := pkgPostgres.WithScoping(searcher)
	prioritySortedSearcher := sorted.Searcher(scopedSearcher, search.NamespacePriority, namespaceRanker)
	// This is currently required due to the priority searcher
	paginatedSearcher := paginated.Paginated(prioritySortedSearcher)
	return paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
}

func storageNamespaceToNamespaceForSAC(ns *storage.NamespaceMetadata) *namespaceForSAC {
	return &namespaceForSAC{
		ID:          ns.GetId(),
		name:        ns.GetName(),
		clusterName: ns.GetClusterName(),
		labels:      ns.GetLabels(),
	}
}

type namespaceForSAC struct {
	ID          string
	name        string
	clusterName string
	labels      map[string]string
}

func (n *namespaceForSAC) GetID() string {
	if n == nil {
		return ""
	}
	return n.ID
}

func (n *namespaceForSAC) GetName() string {
	if n == nil {
		return ""
	}
	return n.name
}

func (n *namespaceForSAC) GetClusterName() string {
	if n == nil {
		return ""
	}
	return n.clusterName
}

func (n *namespaceForSAC) GetLabels() map[string]string {
	if n == nil {
		return nil
	}
	return n.labels
}
