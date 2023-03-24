package sac

import (
	"context"
	"math"

	bleveSearchLib "github.com/blevesearch/bleve/search"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// SearchHelper facilitates applying scoped access control to search operations.
type SearchHelper interface {
	FilteredSearcher(searcher blevesearch.UnsafeSearcher) search.Searcher
}

// searchResultsChecker is responsible for checking whether a single search result is allowed to be seen.
type searchResultsChecker interface {
	TryAllowed(resourceSC ScopeChecker, resultFields map[string]interface{}) bool
	SearchFieldLabels() []search.FieldLabel
	BleveHook(ctx context.Context, resourceChecker ScopeChecker) blevesearch.HookForCategory
}

type searchHelper struct {
	resource permissions.Resource

	resultsChecker searchResultsChecker

	scopeCheckerFactory scopeCheckerFactory
}

// scopeCheckerFactory will be called to create a ScopeChecker.
type scopeCheckerFactory = func(ctx context.Context, am storage.Access, keys ...ScopeKey) ScopeChecker

// NewSearchHelper returns a new search helper for the given resource.
func NewSearchHelper(resourceMD permissions.ResourceMetadata, optionsMap search.OptionsMap,
	factory scopeCheckerFactory) (SearchHelper, error) {
	var nsScope bool

	switch resourceMD.GetScope() {
	case permissions.GlobalScope:
		return nil, errors.New("search helper cannot be used with globally-scoped resources")
	case permissions.ClusterScope:
		nsScope = false
	case permissions.NamespaceScope:
		nsScope = true
	default:
		return nil, errors.Errorf("unknown resource scope %v", resourceMD.GetScope())
	}

	resultsChecker, err := newClusterNSFieldBaseResultsChecker(optionsMap, nsScope)

	if err != nil {
		return nil, errors.Wrapf(err, "creating search helper for resource %v", resourceMD)
	}

	return &searchHelper{
		resource:            resourceMD.GetResource(),
		resultsChecker:      resultsChecker,
		scopeCheckerFactory: factory,
	}, nil
}

// FilteredSearcher takes in an unsafe searcher and makes it safe.
func (h *searchHelper) FilteredSearcher(searcher blevesearch.UnsafeSearcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			return h.executeSearch(ctx, q, searcher)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return h.executeCount(ctx, q, searcher)
		},
	}
}

func (h *searchHelper) executeSearch(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) ([]search.Result, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if scopeChecker.IsAllowed() {
		return searcher.Search(ctx, q)
	}

	// Make sure the cluster and perhaps namespace fields are part of the returned fields.
	fieldQB := search.NewQueryBuilder()
	for _, fieldLabel := range h.resultsChecker.SearchFieldLabels() {
		fieldQB = fieldQB.AddStringsHighlighted(fieldLabel, search.WildcardString)
	}

	queryWithFields := search.ConjunctionQuery(q, fieldQB.ProtoQuery())
	queryWithFields.Pagination = &v1.QueryPagination{
		Limit:       math.MaxInt32,
		SortOptions: q.GetPagination().GetSortOptions(),
	}

	var opts []blevesearch.SearchOption
	if hook := h.resultsChecker.BleveHook(ctx, scopeChecker); hook != nil {
		opts = append(opts, blevesearch.WithHook(hook))
	}
	results, err := searcher.Search(ctx, queryWithFields, opts...)
	if err != nil {
		return nil, err
	}

	return h.filterResults(ctx, scopeChecker, results)
}

func (h *searchHelper) executeCount(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) (int, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if scopeChecker.IsAllowed() {
		return searcher.Count(ctx, q)
	}

	results, err := h.executeSearch(ctx, q, searcher)
	return len(results), err
}

func filterDocs(_ context.Context, resultsChecker searchResultsChecker, resourceScopeChecker ScopeChecker, results []*bleveSearchLib.DocumentMatch) ([]*bleveSearchLib.DocumentMatch, error) {
	var allowed []*bleveSearchLib.DocumentMatch
	for _, result := range results {
		if resultsChecker.TryAllowed(resourceScopeChecker, result.Fields) {
			allowed = append(allowed, result)
		}
	}

	return allowed, nil
}

func (h *searchHelper) filterResults(_ context.Context, resourceScopeChecker ScopeChecker, results []search.Result) ([]search.Result, error) {
	var allowed []search.Result
	for _, result := range results {
		if h.resultsChecker.TryAllowed(resourceScopeChecker, result.Fields) {
			allowed = append(allowed, result)
		}
	}

	return allowed, nil
}

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

func (h *pgSearchHelper) FilteredSearcher(searcher blevesearch.UnsafeSearcher) search.Searcher {
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

func (h *pgSearchHelper) executeSearch(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) ([]search.Result, error) {
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

	var opts []blevesearch.SearchOption
	results, err := searcher.Search(ctx, scopedQuery, opts...)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (h *pgSearchHelper) executeCount(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) (int, error) {
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

// searchHelper implementations

type linkedFieldResultsChecker struct {
	linkedCategory          v1.SearchCategory
	linkedResultsChecker    *clusterNSFieldBasedResultsChecker
	internalHighlightFields []string
}

func newLinkedFieldResultsChecker(linkedCategory v1.SearchCategory, linkedResultsChecker *clusterNSFieldBasedResultsChecker) *linkedFieldResultsChecker {
	internalHighlightFields := make([]string, 0, 2)
	internalHighlightFields = append(internalHighlightFields, linkedResultsChecker.clusterIDFieldPath)
	if linkedResultsChecker.namespaceFieldPath != "" {
		internalHighlightFields = append(internalHighlightFields, linkedResultsChecker.namespaceFieldPath)
	}

	return &linkedFieldResultsChecker{
		linkedCategory:          linkedCategory,
		linkedResultsChecker:    linkedResultsChecker,
		internalHighlightFields: internalHighlightFields,
	}
}

func (c *linkedFieldResultsChecker) BleveHook(ctx context.Context, resourceChecker ScopeChecker) blevesearch.HookForCategory {
	linkedCategoryHook := &blevesearch.Hook{
		InternalHighlightFields: c.internalHighlightFields,
		ResultsFilter: func(unfilteredResults []*bleveSearchLib.DocumentMatch) ([]*bleveSearchLib.DocumentMatch, error) {
			filtered, err := filterDocs(ctx, c.linkedResultsChecker, resourceChecker, unfilteredResults)
			if err != nil {
				return nil, err
			}
			return filtered, nil
		},
	}

	mainHook := &blevesearch.Hook{}
	mainHook.SubQueryHooks = func(category v1.SearchCategory) *blevesearch.Hook {
		if category == c.linkedCategory {
			return linkedCategoryHook
		}
		return mainHook
	}

	return mainHook.SubQueryHooks
}

func (c *linkedFieldResultsChecker) TryAllowed(_ ScopeChecker, _ map[string]interface{}) bool {
	// We allow everything, since the linked field checker is responsible for denying.
	return true
}

func (c *linkedFieldResultsChecker) SearchFieldLabels() []search.FieldLabel {
	return c.linkedResultsChecker.SearchFieldLabels()
}

// clusterNSFieldBasedResultsChecker inspects the `Cluster ID` and optionally the `Namespace`
// field of search results, to determine whether the principal performing the search is allowed
// to see an object.
type clusterNSFieldBasedResultsChecker struct {
	clusterIDFieldPath string
	namespaceFieldPath string
}

func newClusterNSFieldBaseResultsChecker(opts search.OptionsMap, namespaceScoped bool) (searchResultsChecker, error) {
	origMap := opts.Original()
	clusterIDField := origMap[search.ClusterID]
	if clusterIDField == nil {
		return nil, errors.Errorf("field %v not found", search.ClusterID)
	}
	if !clusterIDField.GetStore() {
		return nil, errors.Errorf("field %s is not stored, which is a requirement for access scope enforcement", clusterIDField.GetFieldPath())
	}

	var nsField *search.Field
	if namespaceScoped {
		nsField = origMap[search.Namespace]
		if nsField == nil {
			return nil, errors.Errorf("field %v not found", search.Namespace)
		}
		if !nsField.GetStore() {
			return nil, errors.Errorf("field %s is not stored, which is a requirement for access scope enforcement", nsField.GetFieldPath())
		}

		if nsField.GetCategory() != clusterIDField.GetCategory() {
			return nil, errors.Errorf("namespace field %s is in category %v, while cluster ID field %s is in category %v; this is unsupported", nsField.GetFieldPath(), nsField.GetCategory(), clusterIDField.GetFieldPath(), clusterIDField.GetCategory())
		}
	}

	checker := &clusterNSFieldBasedResultsChecker{
		clusterIDFieldPath: clusterIDField.GetFieldPath(),
		namespaceFieldPath: nsField.GetFieldPath(),
	}

	if clusterIDField.GetCategory() == opts.PrimaryCategory() {
		return checker, nil
	}
	return newLinkedFieldResultsChecker(clusterIDField.GetCategory(), checker), nil
}

func (c *clusterNSFieldBasedResultsChecker) TryAllowed(resourceSC ScopeChecker, resultFields map[string]interface{}) bool {
	key := make([]ScopeKey, 0, 2)
	clusterID, _ := resultFields[c.clusterIDFieldPath].(string)
	key = append(key, ClusterScopeKey(clusterID))
	if c.namespaceFieldPath != "" {
		namespace, _ := resultFields[c.namespaceFieldPath].(string)
		key = append(key, NamespaceScopeKey(namespace))
	}
	return resourceSC.IsAllowed(key...)
}

func (c *clusterNSFieldBasedResultsChecker) SearchFieldLabels() []search.FieldLabel {
	fieldLabels := make([]search.FieldLabel, 0, 2)
	fieldLabels = append(fieldLabels, search.ClusterID)
	if c.namespaceFieldPath != "" {
		fieldLabels = append(fieldLabels, search.Namespace)
	}
	return fieldLabels
}

func (c *clusterNSFieldBasedResultsChecker) BleveHook(context.Context, ScopeChecker) blevesearch.HookForCategory {
	return nil
}
