package sac

import (
	"context"
	"math"
	"strings"
	"time"

	bleveSearchLib "github.com/blevesearch/bleve/search"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/postgres"
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
	resource permissions.ResourceMetadata

	resultsChecker searchResultsChecker

	optionsMap search.OptionsMap
}

// NewSearchHelper returns a new search helper for the given resource.
func NewSearchHelper(resourceMD permissions.ResourceMetadata, optionsMap search.OptionsMap) (SearchHelper, error) {
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
		resource:       resourceMD,
		resultsChecker: resultsChecker,
		optionsMap:     optionsMap,
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

func (h *searchHelper) executeSearch(ctx context.Context, q *v1.Query, searcher blevesearch.UnsafeSearcher) ([]search.Result /*, [][]byte*/, error) {
	scopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(h.resource)
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
	scopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(h.resource)
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

// postgresql implementation

type pgSearchHelper struct {
	resource permissions.ResourceMetadata

	resultsChecker searchResultsChecker

	optionsMap search.OptionsMap

	searchCategory v1.SearchCategory

	statsLabel string

	pool *pgxpool.Pool
}

func NewPgSearchHelper(resourceMD permissions.ResourceMetadata, optionsMap search.OptionsMap, searchCategory v1.SearchCategory, statsLabel string, pool *pgxpool.Pool) (search.Searcher, error) {
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

	return &pgSearchHelper{
		resource:       resourceMD,
		resultsChecker: resultsChecker,
		optionsMap:     optionsMap,
		searchCategory: searchCategory,
		statsLabel:     statsLabel,
		pool:           pool,
	}, nil
}

func exactSearchString(searchString string) string {
	var builder strings.Builder
	builder.WriteByte('"')
	builder.WriteString(searchString)
	builder.WriteByte('"')
	return builder.String()
}

func appendSearchQuery(queries []*v1.Query, searchField search.FieldLabel, searchValue string) []*v1.Query {
	subQuery := search.NewQueryBuilder().AddStrings(searchField, exactSearchString(searchValue))
	return append(queries, subQuery.ProtoQuery())
}

func buildClusterLevelSACQueryFilter(sacTree *effectiveaccessscope.ScopeTree) *v1.Query {
	if sacTree == nil {
		return nil
	}
	if sacTree.State != effectiveaccessscope.Partial {
		return nil
	}
	clusterIDs := sacTree.GetClusterIDs()
	clusterQueries := make([]*v1.Query, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusterAccessScope := sacTree.GetClusterByID(clusterID)
		if clusterAccessScope.State == effectiveaccessscope.Included {
			clusterQueries = appendSearchQuery(clusterQueries, search.ClusterID, clusterID)
		}
	}
	if len(clusterQueries) == 0 {
		return nil
	}
	return search.DisjunctionQuery(clusterQueries...)
}

func buildClusterNamespaceLevelSACQueryFilter(sacTree *effectiveaccessscope.ScopeTree) *v1.Query {
	if sacTree == nil {
		return nil
	}
	if sacTree.State != effectiveaccessscope.Partial {
		return nil
	}
	clusterIDs := sacTree.GetClusterIDs()
	clusterQueries := make([]*v1.Query, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		clusterAccessScope := sacTree.GetClusterByID(clusterID)
		if clusterAccessScope.State == effectiveaccessscope.Included {
			clusterQueries = appendSearchQuery(clusterQueries, search.ClusterID, clusterID)
		} else if clusterAccessScope.State == effectiveaccessscope.Partial {
			clusterSearchID := exactSearchString(clusterID)
			clusterMatchQuery := search.NewQueryBuilder().AddStrings(search.ClusterID, clusterSearchID)
			namespaceQueries := make([]*v1.Query, 0, len(clusterAccessScope.Namespaces))
			for namespaceName, namespaceAccessScope := range clusterAccessScope.Namespaces {
				if namespaceAccessScope.State == effectiveaccessscope.Included {
					namespaceQueries = appendSearchQuery(namespaceQueries, search.Namespace, namespaceName)
				}
			}
			if len(namespaceQueries) > 0 {
				namespaceSubQuery := search.DisjunctionQuery(namespaceQueries...)
				clusterSubQuery := search.ConjunctionQuery(clusterMatchQuery.ProtoQuery(), namespaceSubQuery)
				clusterQueries = append(clusterQueries, clusterSubQuery)
			}
		}
	}
	if len(clusterQueries) == 0 {
		return nil
	}
	return search.DisjunctionQuery(clusterQueries...)
}

func buildSACQueryFilter(sacTree *effectiveaccessscope.ScopeTree, scopeLevel permissions.ResourceScope) *v1.Query {
	switch scopeLevel {
	case permissions.ClusterScope:
		return buildClusterLevelSACQueryFilter(sacTree)
	case permissions.NamespaceScope:
		return buildClusterNamespaceLevelSACQueryFilter(sacTree)
	default:
		return nil
	}
	return nil
}

func (h *pgSearchHelper) Search(ctx context.Context, q *v1.Query) ([]search.Result /*, [][]byte*/, error) {
	scopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(h.resource)
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
		defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, h.statsLabel)
		results, err := postgres.RunSearchRequest(h.searchCategory, q, h.pool, h.optionsMap)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
		return results, nil
	}

	//4. Generate where clause from merged EASes and resource (easy part for Clusters, namespaces etc and hard part images, cves etc)
	//eas, err := scopeChecker.EffectiveAccessScope(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//sacQB := search.NewQueryBuilder()
	//switch h.resource.GetScope() {
	//case permissions.GlobalScope:
	//	return nil, errors.New("Effective Access Scope has no sense with globally-scoped resources")
	//case permissions.ClusterScope:
	//	nsScope = false
	//case permissions.NamespaceScope:
	//	nsScope = true
	//default:
	//	return nil, errors.Errorf("unknown resource scope %v", h.resource.GetScope())
	//}

	//if features.PostgresPOC.Enabled() {
	//	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.SearchAndGet, "ListAlert")
	/* 5. Append clause to SQL query */
	//rows, err := postgres.RunSearchRequestValue(h.optionsMap.PrimaryCategory(), q, globaldb.GetPostgresDB(), h.optionsMap)
	//	if err != nil {
	//		if err == pgx.ErrNoRows {
	//			return nil, nil
	//		}
	//		return nil, err
	//	}
	//	defer rows.Close()
	//	var elems []*storage.ListAlert

	//	for rows.Next() {
	//		var id string
	//		var data []byte
	//		if err := rows.Scan(&id, &data); err != nil {
	//			return nil, err
	//		}
	//		msg := new(storage.Alert)
	//		buf := bytes.NewReader(data)
	//		t := time.Now()
	//		if err := jsonpb.Unmarshal(buf, msg); err != nil {
	//			return nil, err
	//		}
	//		metrics.SetJSONPBOperationDurationTime(t, "Unmarshal", "Alert")
	//		elems = append(elems, convert.AlertToListAlert(msg))
	//	}
	//	return elems, nil
	//}

	sacSearchQuery := q
	if scopeChecker.NeedsPostFiltering() {
		// Make sure the cluster and perhaps namespace fields are part of the returned fields.
		fieldQB := search.NewQueryBuilder()
		for _, fieldLabel := range h.resultsChecker.SearchFieldLabels() {
			fieldQB = fieldQB.AddStringsHighlighted(fieldLabel, search.WildcardString)
		}
		sacSearchQuery = search.ConjunctionQuery(q, fieldQB.ProtoQuery())
	} else {
		// Compute and inject scope query filter
		eas, scopeErr := scopeChecker.EffectiveAccessScope(ctx)
		if scopeErr != nil {
			return nil, scopeErr
		}
		if eas.State == effectiveaccessscope.Excluded {
			return nil, nil
		} else if eas.State == effectiveaccessscope.Partial {
			sacQueryFilter := buildSACQueryFilter(eas, h.resource.GetScope())
			if sacQueryFilter != nil {
				sacSearchQuery = search.ConjunctionQuery(q, sacQueryFilter)
			}
		}
	}
	sacSearchQuery.Pagination = &v1.QueryPagination{
		Limit:       math.MaxInt32,
		SortOptions: q.GetPagination().GetSortOptions(),
	}

	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Search, h.statsLabel)
	results, err := postgres.RunSearchRequest(h.searchCategory, sacSearchQuery, h.pool, h.optionsMap)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if scopeChecker.NeedsPostFiltering() {
		return h.filterResults(ctx, scopeChecker, results)
	}
	return results, nil
}

func (h *pgSearchHelper) Count(ctx context.Context, q *v1.Query) (int, error) {
	scopeChecker := GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(h.resource)
	if ok, err := scopeChecker.Allowed(ctx); err != nil {
		return 0, err
	} else if ok {
		defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, h.statsLabel)
		count, err := postgres.RunCountRequest(h.searchCategory, q, h.pool, h.optionsMap)
		if err != nil {
			if err == pgx.ErrNoRows {
				return 0, nil
			}
			return 0, err
		}
		return count, nil
	}

	defer metrics.SetIndexOperationDurationTime(time.Now(), ops.Count, h.statsLabel)
	results, err := h.Search(ctx, q)
	return len(results), err
}

func (h *pgSearchHelper) filterResultsOnce(resourceScopeChecker ScopeChecker, results []search.Result) (allowed []search.Result, maybe []search.Result) {
	for _, result := range results {
		resultFields := make(map[string]interface{}, 0)
		for _, filterField := range h.resultsChecker.SearchFieldLabels() {
			field, inOptions := h.optionsMap.Get(filterField.String())
			if !inOptions {
				continue
			}
			fieldPath := field.GetFieldPath()
			_, inMatches := result.Matches[fieldPath]
			if inMatches {
				matches := result.Matches[fieldPath]
				if len(matches) > 0 {
					resultFields[fieldPath] = matches[0]
				}
			}
		}
		if res := h.resultsChecker.TryAllowed(resourceScopeChecker, resultFields); res == Allow {
			allowed = append(allowed, result)
		} else if res == Unknown {
			maybe = append(maybe, result)
		}
	}
	return
}

func (h *pgSearchHelper) filterResults(ctx context.Context, resourceScopeChecker ScopeChecker, results []search.Result) ([]search.Result, error) {
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
