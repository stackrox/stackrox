package sac

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
)

// SearchHelper facilitates applying scoped access control to search operations.
type SearchHelper interface {
	FilteredSearcher(searcher search.Searcher) search.Searcher
}

// scopeCheckerFactory will be called to create a ScopeChecker.
type scopeCheckerFactory = func(ctx context.Context, am storage.Access, keys ...ScopeKey) ScopeChecker

// Postgres implementation

type pgSearchHelper struct {
	resourceMD          permissions.ResourceMetadata
	scopeCheckerFactory scopeCheckerFactory
}

// NewPgSearchHelper returns a new search helper for the given resource.
func NewPgSearchHelper(resourceMD permissions.ResourceMetadata, factory scopeCheckerFactory) (SearchHelper, error) {
	if resourceMD.GetScope() == permissions.GlobalScope {
		return nil, errors.New("search helper cannot be used with globally-scoped resources")
	}
	resourceScope := resourceMD.GetScope()
	if resourceScope != permissions.NamespaceScope && resourceScope != permissions.ClusterScope {
		return nil, errors.Errorf("unknown resource scope %v", resourceMD.GetScope())
	}
	return &pgSearchHelper{
		resourceMD:          resourceMD,
		scopeCheckerFactory: factory,
	}, nil
}

func (h *pgSearchHelper) FilteredSearcher(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			return h.executeSearch(ctx, q, searcher)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return h.executeCount(ctx, q, searcher)
		},
	}
}

func (h *pgSearchHelper) enrichQueryWithSACFilter(effectiveAccessScope *effectiveaccessscope.ScopeTree, q *v1.Query) (*v1.Query, error) {
	var sacQueryFilter *v1.Query
	var err error

	// Build SAC filter
	switch h.resourceMD.GetScope() {
	case permissions.NamespaceScope:
		sacQueryFilter, err = BuildNonVerboseClusterNamespaceLevelSACQueryFilter(effectiveAccessScope)
		if err != nil {
			return nil, err
		}
	case permissions.ClusterScope:
		sacQueryFilter, err = BuildNonVerboseClusterLevelSACQueryFilter(effectiveAccessScope)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("Invalid scope %v for resource %v", h.resourceMD.GetScope(), h.resourceMD)
	}
	scopedQuery := search.FilterQueryByQuery(q, sacQueryFilter)

	return scopedQuery, nil
}

func (h *pgSearchHelper) executeSearch(ctx context.Context, q *v1.Query, searcher search.Searcher) ([]search.Result, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if scopeChecker.IsAllowed() {
		return searcher.Search(ctx, q)
	}

	resourceWithAccess := permissions.View(h.resourceMD)
	effectiveAccessScope, err := scopeChecker.EffectiveAccessScope(resourceWithAccess)
	if err != nil {
		return nil, err
	}
	scopedQuery, err := h.enrichQueryWithSACFilter(effectiveAccessScope, q)
	if err != nil {
		return nil, err
	}

	results, err := searcher.Search(ctx, scopedQuery)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (h *pgSearchHelper) executeCount(ctx context.Context, q *v1.Query, searcher search.Searcher) (int, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if scopeChecker.IsAllowed() {
		return searcher.Count(ctx, q)
	}

	resourceWithAccess := permissions.View(h.resourceMD)
	effectiveAccessScope, err := scopeChecker.EffectiveAccessScope(resourceWithAccess)
	if err != nil {
		return 0, err
	}
	scopedQuery, err := h.enrichQueryWithSACFilter(effectiveAccessScope, q)
	if err != nil {
		return 0, err
	}

	return searcher.Count(ctx, scopedQuery)
}
