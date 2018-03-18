package blevesearch

import (
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search"
	"github.com/blevesearch/bleve/search/query"
)

func transformFields(fields map[string]*v1.SearchRequest_Values, objectMap map[string]string) map[string]*v1.SearchRequest_Values {
	newMap := make(map[string]*v1.SearchRequest_Values, len(fields))
	for k, v := range fields {
		// first field
		spl := strings.SplitN(k, ".", 2)
		transformed, ok := objectMap[spl[0]]
		if !ok {
			newMap[k] = v
			continue
		}
		// this implies that the field is a top level object of this struct
		if transformed == "" {
			newMap[spl[1]] = v
		} else {
			newMap[transformed+"."+spl[1]] = v
		}
	}
	return newMap
}

func collapseHitsIntoStringSlice(hits search.DocumentMatchCollection) (matches []string) {
	matches = make([]string, 0, len(hits))
	for _, match := range hits {
		matches = append(matches, match.ID)
	}
	return
}

func newMatchQuery(field, text string) query.Query {
	matchQuery := bleve.NewMatchPhraseQuery(text)
	matchQuery.SetField(field)
	return matchQuery
}

func valuesToDisjunctionQuery(field string, values *v1.SearchRequest_Values) query.Query {
	disjunctionQuery := bleve.NewDisjunctionQuery()
	for _, v := range values.GetValues() {
		disjunctionQuery.AddQuery(newMatchQuery(field, v))
	}
	return disjunctionQuery
}

func fieldsToQuery(fieldMap map[string]*v1.SearchRequest_Values, objectMap map[string]string) query.Query {
	newFieldMap := transformFields(fieldMap, objectMap)
	conjunctionQuery := bleve.NewConjunctionQuery()
	for field, values := range newFieldMap {
		conjunctionQuery.AddQuery(valuesToDisjunctionQuery(field, values))
	}
	return conjunctionQuery
}

func getScopesQuery(scopes []*v1.Scope, scopeToQuery func(scope *v1.Scope) *query.ConjunctionQuery) *query.DisjunctionQuery {
	if len(scopes) != 0 {
		disjunctionQuery := bleve.NewDisjunctionQuery()
		for _, scope := range scopes {
			// Check if nil as some resources may not be applicable to scopes
			if q := scopeToQuery(scope); q != nil {
				disjunctionQuery.AddQuery(scopeToQuery(scope))
			}
		}
		return disjunctionQuery
	}
	return nil
}

func buildQuery(request *v1.SearchRequest, scopeToQuery func(scope *v1.Scope) *query.ConjunctionQuery, objectMap map[string]string) *query.ConjunctionQuery {
	conjunctionQuery := bleve.NewConjunctionQuery()
	if scopesQuery := getScopesQuery(request.GetScopes(), scopeToQuery); scopesQuery != nil {
		conjunctionQuery.AddQuery(scopesQuery)
	}
	if request.GetFields() != nil || len(request.GetFields()) != 0 {
		conjunctionQuery.AddQuery(fieldsToQuery(request.Fields, objectMap))
	}
	return conjunctionQuery
}

func runSearchRequest(request *v1.SearchRequest, index bleve.Index, scopeToQuery func(scope *v1.Scope) *query.ConjunctionQuery, objectMap map[string]string) ([]string, error) {
	conjunctionQuery := buildQuery(request, scopeToQuery, objectMap)
	return runQuery(conjunctionQuery, index)
}

func runQuery(query query.Query, index bleve.Index) ([]string, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = 50
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	return collapseHitsIntoStringSlice(searchResult.Hits), nil
}
