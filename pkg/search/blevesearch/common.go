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

func newRelationship(src, dst v1.SearchCategory) relationship {
	return relationship{
		src: src,
		dst: dst,
	}
}

// This is a map of category A -> category C and the next hop
var links = map[relationship]v1.SearchCategory{
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES): v1.SearchCategory_IMAGES,
}

var categoryRelationships = map[relationship]*v1.SearchField{
	newRelationship(v1.SearchCategory_DEPLOYMENTS, v1.SearchCategory_IMAGES): {
		FieldPath: "deployment.containers.image.name.sha",
		Type:      v1.SearchDataType_SEARCH_STRING,
	},
}

func runSubQuery(index bleve.Index, category v1.SearchCategory, searchField *v1.SearchField, searchValues []string) (query.Query, error) {
	// Base case is that the category you are looking for is the search field category so just get a "normal search query"
	if category == searchField.GetCategory() {
		return evaluateQuery(category, searchField, searchValues)
	}
	// Get the map of result -> next link
	nextHopCategory, ok := links[newRelationship(category, searchField.GetCategory())]
	if !ok {
		return nil, fmt.Errorf("No link specification for category '%s'", category)
	}

	// Go get the query that needs to be run
	subQuery, err := runSubQuery(index, nextHopCategory, searchField, searchValues)
	if err != nil {
		return nil, err
	}

	results, err := RunQuery(subQuery, index)
	if err != nil {
		return nil, err
	}

	// Gets the relationship. e.g. image.name.sha or deployment.containers.id
	relationshipField, ok := categoryRelationships[newRelationship(category, nextHopCategory)]
	if !ok {
		return nil, fmt.Errorf("No relationship field specified for '%s' to '%s'", category, nextHopCategory)
	}

	parentCategory := nextHopCategory
	// if the next hop was the final one, then the current category is the parent
	if nextHopCategory == searchField.GetCategory() {
		parentCategory = category
	}

	// values is populated with the id of the results, which is used to correlate between parents and their children
	values := make([]string, 0, len(results))
	for _, r := range results {
		values = append(values, r.ID)
	}
	// this conjunction query effectively does a join between the refs on the top-level object and the object itself
	conjunctionQuery := bleve.NewConjunctionQuery(typeQuery(parentCategory))
	q, err := evaluateQuery(parentCategory, relationshipField, values)
	if err != nil {
		return nil, err
	}
	conjunctionQuery.AddQuery(q)
	return conjunctionQuery, nil
}

// FieldsToQuery converts a request and the options for the data type to Bleve types
func FieldsToQuery(index bleve.Index, category v1.SearchCategory, request *v1.ParsedSearchRequest, optionsMap map[string]*v1.SearchField) ([]query.Query, error) {
	var queries []query.Query
	for fieldName, field := range request.GetFields() {
		searchField, ok := optionsMap[fieldName]
		if !ok {
			continue
		}
		if searchField.GetCategory() != category {
			q, err := runSubQuery(index, category, searchField, field.GetValues())
			if err != nil {
				return nil, err
			}
			queries = append(queries, q)
			continue
		}
		q, err := evaluateQuery(category, searchField, field.GetValues())
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	return queries, nil
}

// RunSearchRequest builds a query and runs it against the index.
func RunSearchRequest(category v1.SearchCategory, request *v1.ParsedSearchRequest, index bleve.Index, optionsMap map[string]*v1.SearchField) ([]searchPkg.Result, error) {
	que, err := BuildQuery(index, category, request, optionsMap)
	if err != nil {
		return nil, err
	}
	return RunQuery(que, index)
}

// RunQuery runs the actual query and then collapses the results into a simpler format
func RunQuery(query query.Query, index bleve.Index) ([]searchPkg.Result, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return collapseResults(searchResult), nil
}

// BuildQuery builds a query for the input.
func BuildQuery(index bleve.Index, category v1.SearchCategory, request *v1.ParsedSearchRequest, optionsMap map[string]*v1.SearchField) (query.Query, error) {
	queries, err := buildQuery(index, category, request, optionsMap)
	if err != nil {
		return nil, err
	}

	conjunction := bleve.NewConjunctionQuery()
	conjunction.AddQuery(queries...)
	return conjunction, nil
}

func buildQuery(index bleve.Index, category v1.SearchCategory, request *v1.ParsedSearchRequest, optionsMap map[string]*v1.SearchField) ([]query.Query, error) {
	queries, err := FieldsToQuery(index, category, request, optionsMap)
	if err != nil {
		return nil, err
	}
	if request.GetStringQuery() != "" {
		queries = append(queries, NewMatchPhrasePrefixQuery("", request.GetStringQuery()))
	}
	return queries, nil
}

func typeQuery(category v1.SearchCategory) query.Query {
	q := bleve.NewMatchQuery(category.String())
	q.SetField("type")
	return q
}

func collapseResults(searchResult *bleve.SearchResult) (results []searchPkg.Result) {
	results = make([]searchPkg.Result, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		results = append(results, searchPkg.Result{
			ID:      hit.ID,
			Matches: hit.Fragments,
			Score:   hit.Score,
		})
	}
	return
}
