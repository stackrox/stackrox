package datastore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/index/mappings"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
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
	SearchNamespaces(ctx context.Context, q *v1.Query) ([]*storage.NamespaceMetadata, error)
}

// New returns a new DataStore instance using the provided store and indexer
func New(store store.Store, indexer index.Indexer, deploymentDataStore deploymentDataStore.DataStore) (DataStore, error) {
	ds := &datastoreImpl{
		store:             store,
		indexer:           indexer,
		formattedSearcher: formatSearcher(indexer),
		deployments:       deploymentDataStore,
	}
	if err := ds.buildIndex(); err != nil {
		return nil, err
	}
	if err := ds.initializeRanker(); err != nil {
		return ds, err
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
	deploymentRanker  *ranking.Ranker

	deployments deploymentDataStore.DataStore
}

func (b *datastoreImpl) initializeRanker() error {
	b.namespaceRanker = ranking.NamespaceRanker()
	b.deploymentRanker = ranking.DeploymentRanker()

	return nil
}

func (b *datastoreImpl) buildIndex() error {
	namespaces, err := b.store.GetNamespaces()
	if err != nil {
		return err
	}
	return b.indexer.AddNamespaceMetadatas(namespaces)
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
	b.updateNamespacePriority(namespace)
	return namespace, true, err
}

// GetNamespaces retrieves namespaces matching the request from bolt
func (b *datastoreImpl) GetNamespaces(ctx context.Context) ([]*storage.NamespaceMetadata, error) {
	namespaces, err := b.store.GetNamespaces()
	if err != nil {
		return nil, err
	}

	allowedNamespaces := make([]*storage.NamespaceMetadata, 0, len(namespaces))
	for _, namespace := range namespaces {
		scopeKeys := []sac.ScopeKey{sac.ClusterScopeKey(namespace.GetClusterId()), sac.NamespaceScopeKey(namespace.GetName())}
		if ok, err := namespaceSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS, scopeKeys...).
			Allowed(ctx); err != nil || !ok {
			continue
		}
		allowedNamespaces = append(allowedNamespaces, namespace)
	}

	b.updateNamespacePriority(allowedNamespaces...)
	return allowedNamespaces, nil
}

// AddNamespace adds a namespace to bolt
func (b *datastoreImpl) AddNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := b.store.AddNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespaceMetadata(namespace)
}

// UpdateNamespace updates a namespace to bolt
func (b *datastoreImpl) UpdateNamespace(ctx context.Context, namespace *storage.NamespaceMetadata) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := b.store.UpdateNamespace(namespace); err != nil {
		return err
	}
	return b.indexer.AddNamespaceMetadata(namespace)
}

// RemoveNamespace removes a namespace.
func (b *datastoreImpl) RemoveNamespace(ctx context.Context, id string) error {
	if ok, err := namespaceSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := b.store.RemoveNamespace(id); err != nil {
		return err
	}
	// Remove ranker record here since removal is not handled in risk store as no entry present for namespace
	b.namespaceRanker.Remove(id)

	return b.indexer.DeleteNamespaceMetadata(id)
}

func (b *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return b.formattedSearcher.Search(ctx, q)
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
		b.aggregateDeploymentScores(ns.GetId())
	}
	for _, ns := range nss {
		ns.Priority = b.namespaceRanker.GetRankForID(ns.GetId())
	}
}

func (b *datastoreImpl) aggregateDeploymentScores(namespaceID string) {
	aggregateScore := float32(0.0)
	deploymentReadCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Deployment),
		))

	searchResults, err := b.deployments.Search(deploymentReadCtx,
		search.NewQueryBuilder().
			AddExactMatches(search.NamespaceID, namespaceID).ProtoQuery())
	if err != nil {
		log.Error("deployment search for namespace risk calculation failed")
		return
	}

	for _, r := range searchResults {
		aggregateScore += b.deploymentRanker.GetScoreForID(r.ID)
	}
	b.namespaceRanker.Add(namespaceID, aggregateScore)
}

// Helper functions which format our searching.
///////////////////////////////////////////////

func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	filteredSearcher := namespaceSACSearchHelper.FilteredSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.

	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
