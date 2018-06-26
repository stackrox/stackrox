package blevesearch

import (
	"fmt"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	searchPkg "bitbucket.org/stack-rox/apollo/pkg/search"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

const maxSearchResponses = 2000

// NewPrefixQuery does something
func NewPrefixQuery(field, prefix string) query.Query {
	prefixQuery := bleve.NewPrefixQuery(strings.ToLower(prefix))
	prefixQuery.SetField(field)
	return prefixQuery
}

// GetScopesQuery does something
func GetScopesQuery(scopes []*v1.Scope, scopeToQuery func(scope *v1.Scope) query.Query) query.Query {
	if len(scopes) != 0 {
		disjunctionQuery := bleve.NewDisjunctionQuery()
		for _, scope := range scopes {
			// Check if nil as some resources may not be applicable to scopes
			disjunctionQuery.AddQuery(scopeToQuery(scope))
		}
		return disjunctionQuery
	}
	return bleve.NewMatchAllQuery()
}

// RunSearchRequest does something
func RunSearchRequest(objType string, request *v1.ParsedSearchRequest, index bleve.Index, scopeToQuery func(scope *v1.Scope) query.Query, objectMap map[string]string) ([]searchPkg.Result, error) {
	conjunctionQuery := bleve.NewConjunctionQuery(typeQuery(objType))
	queries, err := buildQuery(request, scopeToQuery, objectMap)
	if err != nil {
		return nil, err
	}
	conjunctionQuery.AddQuery(queries...)
	return RunQuery(conjunctionQuery, index)
}

// RunQuery does something
func RunQuery(query query.Query, index bleve.Index) ([]searchPkg.Result, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses
	searchRequest.Highlight = bleve.NewHighlight()
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return collapseResults(searchResult), nil
}

// FieldsToQuery does something
func FieldsToQuery(fieldMap map[string]*v1.ParsedSearchRequest_Values, objectMap map[string]string) (*query.ConjunctionQuery, error) {
	newFieldMap := transformFields(fieldMap, objectMap)
	conjunctionQuery := bleve.NewConjunctionQuery()
	for field, queryValues := range newFieldMap {
		queryFunc, ok := datatypeToQueryFunc[queryValues.GetField().GetType()]
		if !ok {
			return nil, fmt.Errorf("Query for type %s is not implemented", queryValues.GetField().GetType())
		}
		conjunct, err := queryFunc(field, queryValues.GetValues())
		if err != nil {
			return nil, err
		}
		conjunctionQuery.AddQuery(conjunct)
	}
	return conjunctionQuery, nil
}

func buildQuery(request *v1.ParsedSearchRequest, scopeToQuery func(scope *v1.Scope) query.Query, objectMap map[string]string) ([]query.Query, error) {
	var queries []query.Query
	queries = append(queries, GetScopesQuery(request.GetScopes(), scopeToQuery))
	if request.GetFields() != nil && len(request.GetFields()) != 0 {
		q, err := FieldsToQuery(request.GetFields(), objectMap)
		if err != nil {
			return nil, err
		}
		queries = append(queries, q)
	}
	if request.GetStringQuery() != "" {
		queries = append(queries, NewPrefixQuery("", request.GetStringQuery()))
	}
	return queries, nil
}

func typeQuery(objType string) query.Query {
	q := bleve.NewMatchQuery(objType)
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

func transformKey(key string, objectMap map[string]string) string {
	spl := strings.SplitN(key, ".", 2)
	transformed, ok := objectMap[spl[0]]
	if !ok {
		return key
	}
	// this implies that the field is a top level object of this struct
	if transformed == "" {
		return spl[1]
	}
	return transformed + "." + spl[1]
}

func valuesToDisjunctionQuery(field string, values *v1.ParsedSearchRequest_Values) query.Query {
	disjunctionQuery := bleve.NewDisjunctionQuery()
	for _, v := range values.GetValues() {
		disjunctionQuery.AddQuery(NewPrefixQuery(field, v))
	}
	return disjunctionQuery
}
