package blevesearch

import (
	"fmt"

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
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES): v1.SearchCategory_IMAGES,
	newRelationship(v1.SearchCategory_IMAGES, v1.SearchCategory_DEPLOYMENTS): v1.SearchCategory_DEPLOYMENTS,
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
}

func getValuesFromFields(field string, m map[string]interface{}) []string {
	val, ok := m[field]
	if !ok {
		return nil
	}
	switch obj := val.(type) {
	case string:
		return []string{obj}
	case []string:
		return obj
	case []interface{}:
		values := make([]string, 0, len(obj))
		for _, v := range obj {
			if s, ok := v.(string); ok {
				values = append(values, s)
			}
		}
		return values
	default:
		logger.Errorf("Unknown type field from index: %T", obj)
	}
	return nil
}

// TODO(viswa): Rename this function
func runSubQuery(index bleve.Index, category v1.SearchCategory, searchField *v1.SearchField, searchValue string) (query.Query, error) {
	// Base case is that the category you are looking for is the search field category so just get a "normal search query"
	if category == searchField.GetCategory() {
		return matchFieldQuery(category, searchField, searchValue)
	}
	// Get the map of result -> next link
	nextHopCategory, ok := links[newRelationship(category, searchField.GetCategory())]
	if !ok {
		return nil, fmt.Errorf("no link specification for category '%s'", category)
	}

	// Go get the query that needs to be run
	subQuery, err := runSubQuery(index, nextHopCategory, searchField, searchValue)
	if err != nil {
		return nil, err
	}

	// Gets the relationship. e.g. image.name.sha or deployment.containers.id
	relationshipField, ok := categoryRelationships[newRelationship(category, nextHopCategory)]
	if !ok {
		return nil, fmt.Errorf("no relationship field specified for '%s' to '%s'", category, nextHopCategory)
	}

	results, err := RunQuery(subQuery, index, relationshipField.dstField)
	if err != nil {
		return nil, err
	}

	parentCategory := nextHopCategory
	// if the next hop was the final one, then the current category is the parent
	if nextHopCategory == searchField.GetCategory() {
		parentCategory = category
	}

	// values is populated with the specified field of the results, which is used to correlate between parents and their children
	var values []string
	for _, r := range results {
		values = append(values, getValuesFromFields(relationshipField.dstField, r.Fields)...)
	}

	disjunctionQuery := bleve.NewDisjunctionQuery()
	for _, v := range values {
		q, err := matchFieldQuery(parentCategory, &v1.SearchField{
			FieldPath: relationshipField.srcField,
			Type:      v1.SearchDataType_SEARCH_STRING,
		}, v)
		if err != nil {
			return nil, fmt.Errorf("computing query for field '%s': %s", v, err)
		}
		disjunctionQuery.AddQuery(q)
	}

	// this conjunction query effectively does a join between the refs on the top-level object and the object itself
	return bleve.NewConjunctionQuery(typeQuery(parentCategory), disjunctionQuery), nil
}

// RunSearchRequest builds a query and runs it against the index.
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, index bleve.Index, optionsMap map[searchPkg.FieldLabel]*v1.SearchField) ([]searchPkg.Result, error) {
	que, err := BuildQuery(index, category, q, optionsMap)
	if err != nil {
		return nil, err
	}
	return RunQuery(que, index)
}

// RunQuery runs the actual query and then collapses the results into a simpler format
func RunQuery(query query.Query, index bleve.Index, fields ...string) ([]searchPkg.Result, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses
	searchRequest.Fields = fields
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return collapseResults(searchResult), nil
}

// BuildQuery builds a bleve query for the input query
// It is okay for the input query to be nil or empty; in this case, a query matching all documents of the given category will be returned.
func BuildQuery(index bleve.Index, category v1.SearchCategory, q *v1.Query, optionsMap map[searchPkg.FieldLabel]*v1.SearchField) (query.Query, error) {
	if q.GetQuery() == nil {
		return typeQuery(category), nil
	}

	bleveQuery, err := protoToBleveQuery(q, category, index, optionsMap)
	if err != nil {
		return nil, fmt.Errorf("converting to bleve query: %s", err)
	}

	// If a non-empty query was passed, but we couldn't find a query, that means that the query is invalid
	// for this category somehow. In this case, we return a query that matches nothing.
	// This behaviour is helpful, for example, in Global Search, where a query that is invalid for a
	// certain category will just return no elements of that category.
	if bleveQuery == nil {
		bleveQuery = bleve.NewMatchNoneQuery()
	}
	return bleveQuery, nil
}

func collapseResults(searchResult *bleve.SearchResult) (results []searchPkg.Result) {
	results = make([]searchPkg.Result, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		results = append(results, searchPkg.Result{
			ID:      hit.ID,
			Matches: hit.Fragments,
			Score:   hit.Score,
			Fields:  hit.Fields,
		})
	}
	return
}
