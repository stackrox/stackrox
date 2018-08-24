package blevesearch

import (
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

const maxSearchResponses = 2000

var logger = logging.LoggerForModule()

// FieldsToQuery converts a request and the options for the data type to Bleve types
func FieldsToQuery(category v1.SearchCategory, request *v1.ParsedSearchRequest, optionsMap map[string]*v1.SearchField) ([]query.Query, error) {
	var queries []query.Query
	for fieldName, field := range request.GetFields() {
		searchField, ok := optionsMap[fieldName]
		if !ok {
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
	que, err := BuildQuery(category, request, optionsMap)
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
func BuildQuery(category v1.SearchCategory, request *v1.ParsedSearchRequest, optionsMap map[string]*v1.SearchField) (query.Query, error) {
	queries, err := buildQuery(category, request, optionsMap)
	if err != nil {
		return nil, err
	}

	conjunction := bleve.NewConjunctionQuery()
	conjunction.AddQuery(queries...)
	return conjunction, nil
}

func buildQuery(category v1.SearchCategory, request *v1.ParsedSearchRequest, optionsMap map[string]*v1.SearchField) ([]query.Query, error) {
	queries, err := FieldsToQuery(category, request, optionsMap)
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
