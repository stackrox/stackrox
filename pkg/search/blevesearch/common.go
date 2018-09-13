package blevesearch

import (
	"fmt"
	"math"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

const maxSearchResponses = 2000

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
		srcField: "deployment.containers.image.name.sha",
		dstField: "image.name.sha",
	},
	newRelationship(v1.SearchCategory_IMAGES, v1.SearchCategory_DEPLOYMENTS): {
		srcField: "image.name.sha",
		dstField: "deployment.containers.image.name.sha",
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
	default:
		logger.Errorf("Unknown type field from index: %T", val)
	}
	return ""
}

func getMatchingValuesFromFields(field string, m map[string]interface{}, matchIndices []uint32) []string {
	val, ok := m[field]
	if !ok {
		return nil
	}
	if asSlice, ok := val.([]interface{}); ok {
		values := make([]string, 0, len(matchIndices))
		for _, index := range matchIndices {
			if int(index) > len(asSlice) {
				logger.Errorf("Match index out of range. Field %s, map: %#v, matchIndices: %#v", field, m, matchIndices)
				continue
			}
			strVal := getValueFromField(asSlice[index])
			if strVal != "" {
				values = append(values, strVal)
			}
		}
		return values
	}

	if len(matchIndices) != 1 || matchIndices[0] != 0 {
		return nil
	}
	strVal := getValueFromField(val)
	if strVal == "" {
		return nil
	}
	return []string{strVal}
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
func resolveMatchFieldQuery(index bleve.Index, category v1.SearchCategory, searchField *v1.SearchField,
	searchValue string, highlightCtx highlightContext) (query.Query, error) {
	// Base case is that the category you are looking for is the search field category so just get a "normal search query"
	if category == searchField.GetCategory() {
		if highlightCtx != nil {
			if !searchField.GetStore() {
				return nil, fmt.Errorf("can't request highlights for search field %+v, because it is not stored", searchField)
			}
			highlightCtx.AddFieldToHighlight(searchField.GetFieldPath())
		}
		return matchFieldQuery(category, searchField, searchValue)
	}
	// Get the map of result -> next link
	nextHopCategory, ok := links[newRelationship(category, searchField.GetCategory())]
	if !ok {
		return nil, fmt.Errorf("no link specification for category '%s'", category)
	}

	parentCategory := nextHopCategory
	// if the next hop was the final one, then the current category is the parent
	if nextHopCategory == searchField.GetCategory() {
		parentCategory = category
	}

	// Gets the relationship. e.g. image.name.sha or deployment.containers.id
	relationshipField, ok := categoryRelationships[newRelationship(category, nextHopCategory)]
	if !ok {
		return nil, fmt.Errorf("no relationship field specified for '%s' to '%s'", category, nextHopCategory)
	}

	// Go get the query that needs to be run
	subQuery, err := resolveMatchFieldQuery(index, nextHopCategory, searchField, searchValue, highlightCtx)
	if err != nil {
		return nil, fmt.Errorf("resolving query with next hop: '%s': %s", nextHopCategory, err)
	}

	results, err := runQuery(subQuery, index, highlightCtx, relationshipField.dstField)
	if err != nil {
		return nil, fmt.Errorf("running sub query to retrieve field %s: %s", relationshipField.dstField, err)
	}

	// We now create a new disjunction query with the specified field of the results,
	// which is used to correlate between parents and their children
	disjunctionQuery := bleve.NewDisjunctionQuery()

	// targetField is used for highlighting. If we've run a child query already, and we want to highlight
	// results, then the result will have matches in its Matches field.
	// We add information to the highlightContext that allows us to translate the results on the relationship field
	// (which is what we will actually do the query on) to results on the field that the user actually queried for.
	var targetField string
	for _, r := range results {
		var targetFieldValue string
		if highlightCtx != nil && len(r.Matches) > 0 {
			if len(r.Matches) > 1 {
				panic(fmt.Sprintf("There should only be 0 or 1 fields in matches, but found more: %#v", r.Matches))
			}
			for f, v := range r.Matches {
				if len(v) == 0 {
					break
				}
				if targetField == "" {
					targetField = f
				}
				if targetField != f {
					panic(fmt.Sprintf("Got different target fields %s and %s, this should never happen", targetField, f))
				}
				targetFieldValue = v[0]
			}
			if targetField != "" {
				highlightCtx.AddTranslatedFieldIfNotExists(relationshipField.srcField, targetField)
			}
		}

		for _, fieldValue := range getValuesFromFields(relationshipField.dstField, r.Fields) {
			var q query.Query
			q, err = matchFieldQuery(parentCategory, &v1.SearchField{
				FieldPath: relationshipField.srcField,
				Type:      v1.SearchDataType_SEARCH_STRING,
			}, fieldValue)
			if err != nil {
				return nil, fmt.Errorf("computing query for field '%s': %s", fieldValue, err)
			}
			disjunctionQuery.AddQuery(q)

			if targetField != "" {
				highlightCtx.AddMappingToFieldTranslator(relationshipField.srcField, targetField, fieldValue, targetFieldValue)
			}
		}
	}

	// this conjunction query effectively does a join between the refs on the top-level object and the object itself
	return bleve.NewConjunctionQuery(typeQuery(parentCategory), disjunctionQuery), nil
}

// RunSearchRequest builds a query and runs it against the index.
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, index bleve.Index, optionsMap map[searchPkg.FieldLabel]*v1.SearchField) ([]searchPkg.Result, error) {
	bleveQuery, highlightContext, err := buildQuery(index, category, q, optionsMap)
	if err != nil {
		return nil, err
	}
	return runQuery(bleveQuery, index, highlightContext)
}

// runQuery runs the actual query and then collapses the results into a simpler format
func runQuery(query query.Query, index bleve.Index, highlightCtx highlightContext, fields ...string) ([]searchPkg.Result, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses

	if len(fields) > 0 {
		searchRequest.Fields = fields
	}
	highlightCtx.ApplyToBleveReq(searchRequest)

	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return collapseResults(searchResult, highlightCtx), nil
}

// buildQuery builds a bleve query for the input query
// It is okay for the input query to be nil or empty; in this case, a query matching all documents of the given category will be returned.
func buildQuery(index bleve.Index, category v1.SearchCategory, q *v1.Query, optionsMap map[searchPkg.FieldLabel]*v1.SearchField) (query.Query, highlightContext, error) {
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
		results = append(results, searchPkg.Result{
			ID:      hit.ID,
			Matches: matchingFields,
			Score:   hit.Score,
			Fields:  hit.Fields,
		})
	}
	return
}
