package blevesearch

import (
	"fmt"
	"math"
	"strconv"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mathutil"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch/validpositions"
	"github.com/stackrox/rox/pkg/utils"
)

// Don't limit the max search responses because then functionality can go wonky as we rely on the indexer for correctness
// Anything that is time sensitive will need to pass a limit as a part of the pagination request
// In general, prioritize correctness over latency
const (
	maxSearchResponses = math.MaxInt64
)

var (
	log = logging.LoggerForModule()

	defaultSubQueryContext = bleveContext{
		pagination: &v1.QueryPagination{},
	}
)

type relationship struct {
	src v1.SearchCategory
	dst v1.SearchCategory
}

type bleveContext struct {
	pagination        *v1.QueryPagination
	renderedSortOrder search.SortOrder
	searchAfter       []string

	hook *Hook
}

func newRelationship(src v1.SearchCategory, dst v1.SearchCategory) relationship {
	return relationship{
		src: src,
		dst: dst,
	}
}

// This is a map of category A -> category C and the next hop
var links = map[relationship]v1.SearchCategory{
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_PROCESS_INDICATORS): v1.SearchCategory_PROCESS_INDICATORS,
}

type join struct {
	srcField string
	dstField string
}

var categoryRelationships = map[relationship]join{
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_PROCESS_INDICATORS): {
		srcField: "deployment.id",
		dstField: "process_indicator.deployment_id",
	},
}

func getValueFromField(val interface{}) string {
	switch val := val.(type) {
	case string:
		return val
	case float64:
		i, f := math.Modf(val)
		// If it's an int, return just the int portion.
		if math.Abs(f) < 1e-3 {
			return fmt.Sprintf("%d", int(i))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return strconv.FormatBool(val)
	default:
		log.Errorf("Unknown type field from index: %T", val)
	}
	return ""
}

func getMatchingValuesFromFields(field string, hit *search.DocumentMatch, validArrayPositions *validpositions.Tree, includeArrayPositions bool) ([]string, []search.ArrayPositions) {
	if validArrayPositions.Empty() {
		return nil, nil
	}

	val, ok := hit.Fields[field]
	if !ok {
		return nil, nil
	}

	arrayPositions, ok := hit.FieldArrayPositions[field]
	if !ok {
		return nil, nil
	}

	if asSlice, ok := val.([]interface{}); ok {
		var values []string
		var matchingArrayPositions []search.ArrayPositions
		for i, v := range asSlice {
			if i >= len(arrayPositions) {
				break
			}
			if validArrayPositions.Contains(arrayPositions[i]) {
				strVal := getValueFromField(v)
				values = append(values, strVal)
				if includeArrayPositions {
					matchingArrayPositions = append(matchingArrayPositions, arrayPositions[i])
				}
			}
		}
		return values, matchingArrayPositions
	}

	// This is a singleton element.
	// If it doesn't match, don't return it.
	if len(arrayPositions) != 1 || !validArrayPositions.Contains(arrayPositions[0]) {
		return nil, nil
	}
	strVal := getValueFromField(val)
	if strVal == "" {
		return nil, nil
	}
	return []string{strVal}, arrayPositions
}

type searchFieldAndValue struct {
	sf        *searchPkg.Field
	value     string
	highlight bool
}

// GetValuesFromFields returns the values from the given field as a string slice.
func GetValuesFromFields(field string, m map[string]interface{}) []string {
	val, ok := m[field]
	if !ok {
		return nil
	}

	if asSlice, ok := val.([]interface{}); ok {
		values := make([]string, 0, len(asSlice))
		for _, v := range asSlice {
			strVal := getValueFromField(v)
			if strVal != "" {
				values = append(values, strVal)
			}
		}
		return values
	}

	strVal := getValueFromField(val)
	if strVal == "" {
		return nil
	}
	return []string{strVal}
}

func getSubQueryContext(parent bleveContext, category v1.SearchCategory) bleveContext {
	newCtx := defaultSubQueryContext
	if parent.hook != nil && parent.hook.SubQueryHooks != nil {
		newCtx.hook = parent.hook.SubQueryHooks(category)
	}
	return newCtx
}

// resolveMatchFieldQuery returns a query that matches the given searchField to the given value, in the context of category.
// If the category is the same as the searchField's category, then it just returns a direct Bleve query.
// If not, then it actually uses the relationships to run a query, and return results that it can match category against.
// Example: if category is DEPLOYMENT, but we're searching for image tag = "latest", which is a field on images,
// we first run a query for image tag = "latest" on images, and then extract the image ids from matching images.
// The query returned by this function is then a query on deployments, which matches the deployment's image id field against
// all the returned ids from the subquery we ran.
func resolveMatchFieldQuery(ctx bleveContext, index bleve.Index, category v1.SearchCategory, searchFieldsAndValues []searchFieldAndValue, highlightCtx highlightContext) (query.Query, error) {
	// This is a programming error
	if len(searchFieldsAndValues) == 0 {
		panic("Empty slice of searchFieldsAndValues passed to resolveMatchFieldQuery")
	}

	passedFieldsCategory := searchFieldsAndValues[0].sf.GetCategory()

	// Base case is that the category you are looking for is the search field category so just get a "normal search query"
	if category == passedFieldsCategory {
		return matchAllFieldsQuery(ctx, index, category, searchFieldsAndValues, highlightCtx)
	}

	// Get the map of result -> next link
	nextHopCategory, ok := links[newRelationship(category, passedFieldsCategory)]
	if !ok {
		return nil, fmt.Errorf("no link specification for category '%s'", category)
	}

	parentCategory := nextHopCategory
	// if the next hop was the final one, then the current category is the parent
	if nextHopCategory == passedFieldsCategory {
		parentCategory = category
	}

	// Gets the relationship. e.g. image.name.sha or deployment.containers.id
	relationshipField, ok := categoryRelationships[newRelationship(category, nextHopCategory)]
	if !ok {
		return nil, fmt.Errorf("no relationship field specified for '%s' to '%s'", category, nextHopCategory)
	}

	// Go get the query that needs to be run
	subQueryContext := getSubQueryContext(ctx, nextHopCategory)
	subQuery, err := resolveMatchFieldQuery(subQueryContext, index, nextHopCategory, searchFieldsAndValues, highlightCtx)
	if err != nil {
		return nil, errors.Wrapf(err, "resolving query with next hop: '%s'", nextHopCategory)
	}

	results, err := runQuery(subQueryContext, subQuery, index, highlightCtx, relationshipField.dstField)
	if err != nil {
		return nil, errors.Wrapf(err, "running sub query to retrieve field %s", relationshipField.dstField)
	}
	if len(results) == 0 {
		return bleve.NewMatchNoneQuery(), nil
	}

	// We now create a new disjunction query with the specified field of the results,
	// which is used to correlate between parents and their children
	disjunctionQuery := bleve.NewDisjunctionQuery()

	// Reference set is the overall references so we can dedupe if there are many results for the same top level id
	refSet := make(map[string]struct{})
	for _, r := range results {
		fieldValues := GetValuesFromFields(relationshipField.dstField, r.Fields)
		if len(fieldValues) == 0 {
			continue
		}
		for _, fieldValue := range fieldValues {
			if _, ok := refSet[fieldValue]; ok {
				continue
			}
			refSet[fieldValue] = struct{}{}
			q, err := matchFieldQuery(parentCategory, relationshipField.srcField, v1.SearchDataType_SEARCH_STRING, fieldValue)
			if err != nil {
				return nil, errors.Wrapf(err, "computing query for field '%s'", fieldValue)
			}
			disjunctionQuery.AddQuery(q)
		}
		highlightCtx.AddMappings(relationshipField.srcField, fieldValues, r.Matches)
	}

	// this conjunction query effectively does a join between the refs on the top-level object and the object itself
	return bleve.NewConjunctionQuery(typeQuery(parentCategory), disjunctionQuery), nil
}

// RunSearchRequest builds a query and runs it against the index.
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, index bleve.Index, optionsMap searchPkg.OptionsMap, searchOpts ...SearchOption) ([]searchPkg.Result, error) {
	sortOrder, searchAfter, err := getSortOrderAndSearchAfter(q.GetPagination(), optionsMap)
	if err != nil {
		return nil, err
	}

	ctx := bleveContext{
		pagination:        q.GetPagination(),
		renderedSortOrder: sortOrder,
		searchAfter:       searchAfter,
	}

	opts := opts{}
	for _, opt := range searchOpts {
		if err := opt(&opts); err != nil {
			return nil, errors.Wrap(err, "could not apply search option")
		}
	}

	if opts.hook != nil {
		ctx.hook = opts.hook(category)
	}

	bleveQuery, highlightContext, err := buildQuery(ctx, index, category, q, optionsMap)
	if err != nil {
		return nil, err
	}
	return runQuery(ctx, bleveQuery, index, highlightContext)
}

// RunCountRequest builds a query and runs it to get the count of matches against the index.
func RunCountRequest(category v1.SearchCategory, q *v1.Query, index bleve.Index, optionsMap searchPkg.OptionsMap, searchOpts ...SearchOption) (int, error) {
	var ctx bleveContext
	var opts opts
	for _, opt := range searchOpts {
		if err := opt(&opts); err != nil {
			return 0, errors.Wrap(err, "could not apply search option")
		}
	}

	if opts.hook != nil {
		ctx.hook = opts.hook(category)
	}

	bleveQuery, _, err := buildQuery(ctx, index, category, q, optionsMap)
	if err != nil {
		return 0, err
	}
	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = 0

	result, err := index.Search(searchRequest)
	if err != nil {
		return 0, err
	}
	if result.Total > uint64(mathutil.MaxIntVal) {
		return 0, errors.Errorf("the count is out of range, max: %v got: %v", mathutil.MaxIntVal, result.Total)
	}
	return int(result.Total), nil
}

func getSortOrderAndSearchAfter(pagination *v1.QueryPagination, optionsMap searchPkg.OptionsMap) (search.SortOrder, []string, error) {
	if len(pagination.GetSortOptions()) == 0 {
		return nil, nil, nil
	}

	sortOrder := make([]search.SearchSort, 0, len(pagination.GetSortOptions()))

	var searchAfter []string
	searchAfterHasDocID := false
	allowSearchAfter := true

	for _, so := range pagination.GetSortOptions() {
		var sortField search.SearchSort

		if so.GetField() == searchPkg.DocID.String() {
			sortField = &search.SortDocID{
				Desc: so.GetReversed(),
			}
		} else {
			sf, ok := optionsMap.Get(so.GetField())
			if !ok {
				return nil, nil, errors.Errorf("option %q is not a valid search option", so.GetField())
			}
			sortField = &search.SortField{
				Field:   sf.GetFieldPath(),
				Desc:    so.GetReversed(),
				Type:    search.SortFieldAuto,
				Missing: search.SortFieldMissingLast,
			}
		}

		sortOrder = append(sortOrder, sortField)

		if saOpt, ok := so.GetSearchAfterOpt().(*v1.QuerySortOption_SearchAfter); ok {
			if !allowSearchAfter {
				return nil, nil, errors.New("invalid SearchAfter state: SearchAfter values must start from the beginning of SortOptions and must follow without gaps")
			}
			if so.GetField() == searchPkg.DocID.String() {
				searchAfterHasDocID = true
			}
			searchAfter = append(searchAfter, saOpt.SearchAfter)
		} else {
			allowSearchAfter = false
		}
	}

	if len(searchAfter) > 0 && !searchAfterHasDocID {
		// This checks that SearchAfter will have effect when used or returns an error.
		// It appears that Bleve does not have validations for bleve.SearchRequest.SearchAfter. This closes the gap.
		// See https://github.com/blevesearch/bleve/pull/1182#issuecomment-499216058
		return nil, nil, utils.Should(errors.New("total ordering not guaranteed: SortOrder must contain DocID and SearchAfter value for it to ensure there are no ties, otherwise SearchAfter will not produce correct results"))
	}

	return sortOrder, searchAfter, nil
}

func runBleveQuery(ctx bleveContext, query query.Query, index bleve.Index, highlightCtx highlightContext, includeLocations bool, fields ...string) (*bleve.SearchResult, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses
	if ctx.pagination != nil {
		searchRequest.Sort = ctx.renderedSortOrder
		searchRequest.SearchAfter = ctx.searchAfter
	}
	searchRequest.IncludeLocations = includeLocations

	if len(fields) > 0 {
		searchRequest.Fields = fields
	}

	var hookPP *hookPostProcessor
	if ctx.hook != nil {
		if highlightCtx == nil {
			highlightCtx = make(highlightContext)
		}
		hookPP = ctx.hook.apply(highlightCtx)
	}

	highlightCtx.ApplyToBleveReq(searchRequest)

	result, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	if hookPP != nil {
		result, err = hookPP.apply(result)
	}
	return result, err
}

// runQuery runs the actual query and then collapses the results into a simpler format
func runQuery(ctx bleveContext, q query.Query, index bleve.Index, highlightCtx highlightContext, fields ...string) ([]searchPkg.Result, error) {
	searchResult, err := runBleveQuery(ctx, q, index, highlightCtx, false, fields...)
	if err != nil {
		return nil, err
	}
	return collapseResults(searchResult, highlightCtx), nil
}

// buildQuery builds a bleve query for the input query
// It is okay for the input query to be nil or empty; in this case, a query matching all documents of the given category will be returned.
func buildQuery(ctx bleveContext, index bleve.Index, category v1.SearchCategory, q *v1.Query, optionsMap searchPkg.OptionsMap) (query.Query, highlightContext, error) {
	if q.GetQuery() == nil {
		return typeQuery(category), nil, nil
	}

	queryConverter := newQueryConverter(category, index, optionsMap)
	bleveQuery, highlightCtx, err := queryConverter.convert(ctx, q)
	if err != nil {
		return nil, nil, errors.Wrap(err, "converting to bleve query")
	}

	// If a non-empty query was passed, but we couldn't find a query, that means that the query is invalid
	// for this category somehow. In this case, we return a query that matches nothing.
	// This behaviour is helpful, for example, in Global Search, where a query that is invalid for a
	// certain category will just return no elements of that category.
	if bleveQuery == nil {
		bleveQuery = bleve.NewMatchNoneQuery()
	}
	return bleveQuery, highlightCtx, nil
}

func collapseResults(searchResult *bleve.SearchResult, highlightCtx highlightContext) (results []searchPkg.Result) {
	results = make([]searchPkg.Result, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		matchingFields := highlightCtx.ResolveMatches(hit)
		result := searchPkg.Result{
			ID:      hit.ID,
			Matches: matchingFields,
			Score:   hit.Score,
			Fields:  hit.Fields,
		}
		results = append(results, result)
	}
	return
}
