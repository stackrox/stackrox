package sac

import (
	"context"
	"math"

	bleveSearchLib "github.com/blevesearch/bleve/v2/search"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/utils"
)

// SearchHelper facilitates applying scoped access control to search operations.
type SearchHelper interface {
	Apply(searchFunc func(*v1.Query, ...blevesearch.SearchOption) ([]search.Result, error)) func(context.Context, *v1.Query) ([]search.Result, error)
	ApplyCount(searchFunc func(*v1.Query, ...blevesearch.SearchOption) (int, error)) func(context.Context, *v1.Query) (int, error)
	FilteredSearcher(searcher blevesearch.UnsafeSearcher) search.Searcher
}

// searchResultsChecker is responsible for checking whether a single search result is allowed to be seen.
type searchResultsChecker interface {
	TryAllowed(resourceSC ScopeChecker, resultFields map[string]interface{}) TryAllowedResult
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

// Apply takes in a context-less search function, and returns a search function taking in a context and applying
// scoped access control checks for result filtering.
func (h *searchHelper) Apply(rawSearchFunc func(*v1.Query, ...blevesearch.SearchOption) ([]search.Result, error)) func(context.Context, *v1.Query) ([]search.Result, error) {
	return func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		searcher := blevesearch.UnsafeSearcherImpl{
			SearchFunc: rawSearchFunc,
			CountFunc:  nil,
		}
		return h.executeSearch(ctx, q, searcher)
	}
}

// ApplyCount takes in a context-less count function, and returns a count function taking in a context and applying
// scoped access control checks for result filtering.
func (h *searchHelper) ApplyCount(rawCountFunc func(*v1.Query, ...blevesearch.SearchOption) (int, error)) func(context.Context, *v1.Query) (int, error) {
	return func(ctx context.Context, q *v1.Query) (int, error) {
		searcher := blevesearch.UnsafeSearcherImpl{
			SearchFunc: nil,
			CountFunc:  rawCountFunc,
		}
		return h.executeCount(ctx, q, searcher)
	}
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
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return searcher.Search(q)
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
	results, err := searcher.Search(queryWithFields, opts...)
	if err != nil {
		return nil, err
	}

	return h.filterResults(ctx, scopeChecker, results)
}

func (h *searchHelper) executeCount(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) (int, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return searcher.Count(q)
	}

	results, err := h.executeSearch(ctx, q, searcher)
	return len(results), err
}

func filterDocsOnce(resultsChecker searchResultsChecker, resourceScopeChecker ScopeChecker, results []*bleveSearchLib.DocumentMatch) (allowed []*bleveSearchLib.DocumentMatch, maybe []*bleveSearchLib.DocumentMatch) {
	for _, result := range results {
		if res := resultsChecker.TryAllowed(resourceScopeChecker, result.Fields); res == Allow {
			allowed = append(allowed, result)
		} else if res == Unknown {
			maybe = append(maybe, result)
		}
	}
	return
}

func filterDocs(ctx context.Context, resultsChecker searchResultsChecker, resourceScopeChecker ScopeChecker, results []*bleveSearchLib.DocumentMatch) ([]*bleveSearchLib.DocumentMatch, error) {
	allowed, maybe := filterDocsOnce(resultsChecker, resourceScopeChecker, results)
	if len(maybe) > 0 {
		if err := resourceScopeChecker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		extraAllowed, maybe := filterDocsOnce(resultsChecker, resourceScopeChecker, maybe)
		if len(maybe) > 0 {
			utils.Should(errors.Errorf("still %d maybe results after PerformChecks", len(maybe)))
		}
		allowed = append(allowed, extraAllowed...)
	}

	return allowed, nil
}

func (h *searchHelper) filterResultsOnce(resourceScopeChecker ScopeChecker, results []search.Result) (allowed []search.Result, maybe []search.Result) {
	for _, result := range results {
		if res := h.resultsChecker.TryAllowed(resourceScopeChecker, result.Fields); res == Allow {
			allowed = append(allowed, result)
		} else if res == Unknown {
			maybe = append(maybe, result)
		}
	}
	return
}

func (h *searchHelper) filterResults(ctx context.Context, resourceScopeChecker ScopeChecker, results []search.Result) ([]search.Result, error) {
	allowed, maybe := h.filterResultsOnce(resourceScopeChecker, results)
	if len(maybe) > 0 {
		if err := resourceScopeChecker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		extraAllowed, maybe := h.filterResultsOnce(resourceScopeChecker, maybe)
		if len(maybe) > 0 {
			utils.Should(errors.Errorf("still %d maybe results after PerformChecks", len(maybe)))
		}
		allowed = append(allowed, extraAllowed...)
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
	return &pgSearchHelper{
		resourceMD:          resourceMD,
		scopeCheckerFactory: factory,
	}, nil
}

func (h *pgSearchHelper) Apply(rawSearchFunc func(*v1.Query, ...blevesearch.SearchOption) ([]search.Result, error)) func(context.Context, *v1.Query) ([]search.Result, error) {
	return func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		searcher := blevesearch.UnsafeSearcherImpl{
			SearchFunc: rawSearchFunc,
			CountFunc:  nil,
		}
		return h.executeSearch(ctx, q, searcher)
	}
}

func (h *pgSearchHelper) ApplyCount(rawCountFunc func(*v1.Query, ...blevesearch.SearchOption) (int, error)) func(context.Context, *v1.Query) (int, error) {
	return func(ctx context.Context, q *v1.Query) (int, error) {
		searcher := blevesearch.UnsafeSearcherImpl{
			SearchFunc: nil,
			CountFunc:  rawCountFunc,
		}
		return h.executeCount(ctx, q, searcher)
	}
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

func (h *pgSearchHelper) executeSearch(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) ([]search.Result, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return searcher.Search(q)
	}

	// Generate query filter
	resourceWithAccess := permissions.ResourceWithAccess{
		Resource: h.resourceMD,
		Access:   storage.Access_READ_ACCESS,
	}
	effectiveaccessscope, err := scopeChecker.EffectiveAccessScope(resourceWithAccess)
	if err != nil {
		return nil, err
	}
	var sacQueryFilter *v1.Query
	switch h.resourceMD.GetScope() {
	case permissions.NamespaceScope:
		sacQueryFilter, err = BuildClusterNamespaceLevelSACQueryFilter(effectiveaccessscope)
		if err != nil {
			return nil, err
		}
	case permissions.ClusterScope:
		sacQueryFilter, err = BuildClusterLevelSACQueryFilter(effectiveaccessscope)
		if err != nil {
			return nil, err
		}
	default:
		sacQueryFilter = nil
	}
	scopedQuery := search.ConjunctionQuery(q, sacQueryFilter)
	if q == nil {
		scopedQuery.Pagination = &v1.QueryPagination{
			Limit: math.MaxInt32,
		}
	} else {
		scopedQuery.Pagination = &v1.QueryPagination{
			Limit:       math.MaxInt32,
			SortOptions: q.GetPagination().GetSortOptions(),
		}
	}

	var opts []blevesearch.SearchOption
	results, err := searcher.Search(scopedQuery, opts...)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func (h *pgSearchHelper) executeCount(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) (int, error) {
	scopeChecker := h.scopeCheckerFactory(ctx, storage.Access_READ_ACCESS)
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return 0, err
	} else if ok {
		return searcher.Count(q)
	}

	results, err := h.executeSearch(ctx, q, searcher)
	return len(results), err
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

func (c *linkedFieldResultsChecker) TryAllowed(resourceSC ScopeChecker, resultFields map[string]interface{}) TryAllowedResult {
	// We allow everything, since the linked field checker is responsible for denying.
	return Allow
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

func (c *clusterNSFieldBasedResultsChecker) TryAllowed(resourceSC ScopeChecker, resultFields map[string]interface{}) TryAllowedResult {
	key := make([]ScopeKey, 0, 2)
	clusterID, _ := resultFields[c.clusterIDFieldPath].(string)
	key = append(key, ClusterScopeKey(clusterID))
	if c.namespaceFieldPath != "" {
		namespace, _ := resultFields[c.namespaceFieldPath].(string)
		key = append(key, NamespaceScopeKey(namespace))
	}
	return resourceSC.TryAllowed(key...)
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
