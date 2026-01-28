package datastore

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/convert/storagetoeffectiveaccessscope"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/namespace/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sorted"
)

//go:generate mockgen-wrapper

// DataStore provides storage and indexing functionality for namespaces.
type DataStore interface {
	GetNamespace(ctx context.Context, id string) (*storage.NamespaceMetadata, bool, error)
	GetAllNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error)
	GetNamespacesForSAC() ([]effectiveaccessscope.Namespace, error)
	GetManyNamespaces(ctx context.Context, id []string) ([]*storage.NamespaceMetadata, error)

	AddNamespace(context.Context, *storage.NamespaceMetadata) error
	UpdateNamespace(context.Context, *storage.NamespaceMetadata) error
	RemoveNamespace(ctx context.Context, id string) error

	SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store.
func New(nsStore store.Store, deploymentDataStore deploymentDataStore.DataStore, namespaceRanker *ranking.Ranker) DataStore {
	return &datastoreImpl{
		store:           nsStore,
		deployments:     deploymentDataStore,
		namespaceRanker: namespaceRanker,
	}
}

// Constructor of the namespace storage takes a Store object as an argument,
// but it's an internal type which could not be constructed outside. Such
// approach limits the ways how namespace storage could be instantiated to only
// the singleton. This method allows to avoid the limitation.
func NewStorage(db pg.DB) store.Store {
	return pgStore.New(db)
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
			allowedNamespaces = append(allowedNamespaces, namespace.CloneVT())
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(ctx, walkFn); err != nil {
		return nil, err
	}
	b.updateNamespacePriority(allowedNamespaces...)
	return allowedNamespaces, nil
}

// GetNamespacesForSAC retrieves namespaces matching the request
func (b *datastoreImpl) GetNamespacesForSAC() ([]effectiveaccessscope.Namespace, error) {
	return storagetoeffectiveaccessscope.Namespaces(b.store.GetAllFromCacheForSAC()), nil
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

// UpdateNamespace updates a namespace to the database
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
	if q == nil {
		q = search.EmptyQuery()
	}
	// Clone the query and add select fields for SearchResult construction
	clonedQuery := q.CloneVT()

	// Add required fields for SearchResult proto: name (namespace name) and location (cluster/namespace)
	// ForSearchResults will add these as select fields to the query
	selectSelects := []*v1.QuerySelect{
		search.NewQuerySelect(search.Namespace).Proto(),
		search.NewQuerySelect(search.Cluster).Proto(),
	}
	clonedQuery.Selects = append(clonedQuery.GetSelects(), selectSelects...)

	results, err := b.Search(ctx, clonedQuery)
	if err != nil {
		return nil, err
	}

	// Build name and location strings from selected fields
	for i := range results {
		var clusterName string
		var namespaceName string

		// Extract values from FieldValues if available
		// Keys are lowercase versions of the field names (e.g., "cluster", "namespace")
		// Values are already converted to strings by the postgres framework
		if results[i].FieldValues != nil {
			if cluster, ok := results[i].FieldValues[strings.ToLower(search.Cluster.String())]; ok {
				clusterName = cluster
			}
			if namespace, ok := results[i].FieldValues[strings.ToLower(search.Namespace.String())]; ok {
				namespaceName = namespace
			}
		}

		results[i].Name = namespaceName
		results[i].Location = fmt.Sprintf("%s/%s", clusterName, namespaceName)
	}

	// Convert search Results directly to SearchResult protos without a second database pass
	return search.ResultsToSearchResultProtos(results, &NamespaceSearchResultConverter{}), nil
}

func (b *datastoreImpl) searchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, []search.Result, error) {
	// TODO(ROX-29943): remove unnecessary calls to database
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

// NamespaceSearchResultConverter implements search.SearchResultConverter for namespace search results.
// This enables single-pass query construction for SearchResult protos.
type NamespaceSearchResultConverter struct{}

func (c *NamespaceSearchResultConverter) BuildName(result *search.Result) string {
	return result.Name
}

func (c *NamespaceSearchResultConverter) BuildLocation(result *search.Result) string {
	return result.Location
}

func (c *NamespaceSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_NAMESPACES
}
