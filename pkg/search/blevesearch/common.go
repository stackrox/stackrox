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

	results, err := RunQuery(subQuery, index)
	if err != nil {
		return nil, err
	}

	// Gets the relationship. e.g. image.name.sha or deployment.containers.id
	relationshipField, ok := categoryRelationships[newRelationship(category, nextHopCategory)]
	if !ok {
		return nil, fmt.Errorf("no relationship field specified for '%s' to '%s'", category, nextHopCategory)
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

	disjunctionQuery := bleve.NewDisjunctionQuery()
	for _, r := range results {
		q, err := matchFieldQuery(parentCategory, relationshipField, r.ID)
		if err != nil {
			return nil, fmt.Errorf("computing query for ID '%s': %s", r.ID, err)
		}
		disjunctionQuery.AddQuery(q)
	}

	// this conjunction query effectively does a join between the refs on the top-level object and the object itself
	return bleve.NewConjunctionQuery(typeQuery(parentCategory), disjunctionQuery), nil
}

// RunSearchRequest builds a query and runs it against the index.
func RunSearchRequest(category v1.SearchCategory, q *v1.Query, index bleve.Index, optionsMap map[string]*v1.SearchField) ([]searchPkg.Result, error) {
	que, err := BuildQuery(index, category, q, optionsMap)
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

// BuildQuery builds a bleve query for the input query
// It is okay for the input query to be nil or empty; in this case, a query matching all documents of the given category will be returned.
func BuildQuery(index bleve.Index, category v1.SearchCategory, q *v1.Query, optionsMap map[string]*v1.SearchField) (query.Query, error) {
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
		})
	}
	return
}
