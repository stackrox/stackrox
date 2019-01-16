package blevesearch

import (
	"fmt"
	"math"
	"strconv"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch/validpositions"
)

const maxSearchResponses = 20000

var logger = logging.LoggerForModule()

type relationship struct {
	src v1.SearchCategory
	dst v1.SearchCategory
}

func newRelationship(src v1.SearchCategory, dst v1.SearchCategory) relationship {
	return relationship{
		src: src,
		dst: dst,
	}
}

// This is a map of category A -> category C and the next hop
var links = map[relationship]v1.SearchCategory{
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES):             v1.SearchCategory_IMAGES,
	newRelationship(v1.SearchCategory_IMAGES, v1.SearchCategory_DEPLOYMENTS):             v1.SearchCategory_DEPLOYMENTS,
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_PROCESS_INDICATORS): v1.SearchCategory_PROCESS_INDICATORS,
}

type join struct {
	srcField string
	dstField string
}

var categoryRelationships = map[relationship]join{
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES): {
		srcField: "deployment.containers.image.id",
		dstField: "image.id",
	},
	newRelationship(v1.SearchCategory_IMAGES, v1.SearchCategory_DEPLOYMENTS): {
		srcField: "image.id",
		dstField: "deployment.containers.image.id",
	},
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
		logger.Errorf("Unknown type field from index: %T", val)
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
	sf        *v1.SearchField
	value     string
	highlight bool
}

func getValuesFromFields(field string, m map[string]interface{}) []string {
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

// resolveMatchFieldQuery returns a query that matches the given searchField to the given value, in the context of category.
// If the category is the same as the searchField's category, then it just returns a direct Bleve query.
// If not, then it actually uses the relationships to run a query, and return results that it can match category against.
// Example: if category is DEPLOYMENT, but we're searching for image tag = "latest", which is a field on images,
// we first run a query for image tag = "latest" on images, and then extract the image ids from matching images.
// The query returned by this function is then a query on deployments, which matches the deployment's image id field against
// all the returned ids from the subquery we ran.
func resolveMatchFieldQuery(index bleve.Index, category v1.SearchCategory, searchFieldsAndValues []searchFieldAndValue, highlightCtx highlightContext) (query.Query, error) {
	// This is a programming error
	if len(searchFieldsAndValues) == 0 {
		panic("Empty slice of searchFieldsAndValues passed to resolveMatchFieldQuery")
	}

	passedFieldsCategory := searchFieldsAndValues[0].sf.GetCategory()

	// Base case is that the category you are looking for is the search field category so just get a "normal search query"
	if category == passedFieldsCategory {
		return matchAllFieldsQuery(index, category, searchFieldsAndValues, highlightCtx)
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
	subQuery, err := resolveMatchFieldQuery(index, nextHopCategory, searchFieldsAndValues, highlightCtx)
	if err != nil {
		return nil, fmt.Errorf("resolving query with next hop: '%s': %s", nextHopCategory, err)
	}

	results, err := runQuery(subQuery, index, highlightCtx, relationshipField.dstField)
	if err != nil {
		return nil, fmt.Errorf("running sub query to retrieve field %s: %s", relationshipField.dstField, err)
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
		fieldValues := getValuesFromFields(relationshipField.dstField, r.Fields)
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
				return nil, fmt.Errorf("computing query for field '%s': %s", fieldValue, err)
			}
			disjunctionQuery.AddQuery(q)
		}
		highlightCtx.AddMappings(relationshipField.srcField, fieldValues, r.Matches)
	}

	// this conjunction query effectively does a join between the refs on the top-level object and the object itself
	return bleve.NewConjunctionQuery(typeQuery(parentCategory), disjunctionQuery), nil
}

// RunSearchRequest builds a query and runs it against the index.
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, index bleve.Index, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
	bleveQuery, highlightContext, err := buildQuery(index, category, q, optionsMap)
	if err != nil {
		return nil, err
	}
	return runQuery(bleveQuery, index, highlightContext)
}

func runBleveQuery(query query.Query, index bleve.Index, highlightCtx highlightContext, includeLocations bool, fields ...string) (*bleve.SearchResult, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses
	searchRequest.IncludeLocations = includeLocations

	if len(fields) > 0 {
		searchRequest.Fields = fields
	}
	highlightCtx.ApplyToBleveReq(searchRequest)

	return index.Search(searchRequest)
}

// runQuery runs the actual query and then collapses the results into a simpler format
func runQuery(query query.Query, index bleve.Index, highlightCtx highlightContext, fields ...string) ([]searchPkg.Result, error) {
	searchResult, err := runBleveQuery(query, index, highlightCtx, false, fields...)
	if err != nil {
		return nil, err
	}
	return collapseResults(searchResult, highlightCtx), nil
}

// buildQuery builds a bleve query for the input query
// It is okay for the input query to be nil or empty; in this case, a query matching all documents of the given category will be returned.
func buildQuery(index bleve.Index, category v1.SearchCategory, q *v1.Query, optionsMap searchPkg.OptionsMap) (query.Query, highlightContext, error) {
	if q.GetQuery() == nil {
		return typeQuery(category), nil, nil
	}

	queryConverter := newQueryConverter(category, index, optionsMap)
	bleveQuery, highlightCtx, err := queryConverter.convert(q)
	if err != nil {
		return nil, nil, fmt.Errorf("converting to bleve query: %s", err)
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
