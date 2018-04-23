package blevesearch

import (
	"fmt"
	"strconv"
	"strings"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	searchPkg "bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
)

const maxSearchResponses = 100

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

func splitFunc(r rune) bool {
	return r == ' ' || r == '-'
}

func splitByDelimiters(field string) []string {
	return strings.FieldsFunc(field, splitFunc)
}

// These are inexact matches and the allowable distance is dictated by the global fuzziness
func newPrefixQuery(field, prefix string) query.Query {
	// Must split the fields via the spaces
	var conjunction query.ConjunctionQuery
	// todo(cgorman) replace this by MultiPhrasePrefixQuery when it gets merged into master (or we can cherry-pick)
	for _, val := range splitByDelimiters(prefix) {
		val = strings.ToLower(val)
		prefixQuery := bleve.NewPrefixQuery(val)
		prefixQuery.SetField(field)
		conjunction.AddQuery(prefixQuery)
	}
	return &conjunction
}

func valuesToDisjunctionQuery(field string, values *v1.ParsedSearchRequest_Values) query.Query {
	disjunctionQuery := bleve.NewDisjunctionQuery()
	for _, v := range values.GetValues() {
		disjunctionQuery.AddQuery(newPrefixQuery(field, v))
	}
	return disjunctionQuery
}

func getScopesQuery(scopes []*v1.Scope, scopeToQuery func(scope *v1.Scope) query.Query) query.Query {
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

func buildQuery(request *v1.ParsedSearchRequest, scopeToQuery func(scope *v1.Scope) query.Query, objectMap map[string]string) (*query.ConjunctionQuery, error) {
	conjunctionQuery := bleve.NewConjunctionQuery()
	conjunctionQuery.AddQuery(getScopesQuery(request.GetScopes(), scopeToQuery))
	if request.GetFields() != nil && len(request.GetFields()) != 0 {
		q, err := fieldsToQuery(request.GetFields(), objectMap)
		if err != nil {
			return nil, err
		}
		conjunctionQuery.AddQuery(q)
	}
	if request.GetStringQuery() != "" {
		conjunctionQuery.AddQuery(newPrefixQuery("", request.GetStringQuery()))
	}
	return conjunctionQuery, nil
}

func runSearchRequest(request *v1.ParsedSearchRequest, index bleve.Index, scopeToQuery func(scope *v1.Scope) query.Query, objectMap map[string]string) ([]searchPkg.Result, error) {
	conjunctionQuery, err := buildQuery(request, scopeToQuery, objectMap)
	if err != nil {
		return nil, err
	}
	return runQuery(conjunctionQuery, index)
}

func runQuery(query query.Query, index bleve.Index) ([]searchPkg.Result, error) {
	searchRequest := bleve.NewSearchRequest(query)
	// Initial size is 10 which seems small
	searchRequest.Size = maxSearchResponses
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Fields = []string{"*"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		return nil, err
	}
	metrics.SetAPIRequestDurationTime(searchResult.Took)
	return collapseResults(searchResult), nil
}

var datatypeToQueryFunc = map[v1.SearchDataType]func(string, []string) (query.Query, error){
	v1.SearchDataType_SEARCH_STRING:      newStringQuery,
	v1.SearchDataType_SEARCH_BOOL:        newBoolQuery,
	v1.SearchDataType_SEARCH_NUMERIC:     newNumericQuery,
	v1.SearchDataType_SEARCH_SEVERITY:    newSeverityQuery,
	v1.SearchDataType_SEARCH_ENFORCEMENT: newEnforcementQuery,
}

func newStringQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, val := range values {
		d.AddQuery(newPrefixQuery(field, val))
	}
	return d, nil
}

func newBoolQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, val := range values {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, err
		}
		q := bleve.NewBoolFieldQuery(b)
		q.FieldVal = field
		d.AddQuery(q)
	}
	return d, nil
}

func floatPtr(f float64) *float64 {
	return &f
}

func boolPtr(b bool) *bool {
	return &b
}

func parseNumericStringToPtr(s string) (*float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &val, nil
}

func parseNumericValue(num string) (min *float64, max *float64, inclusive *bool, err error) {
	if strings.HasPrefix(num, "<=") {
		inclusive = boolPtr(true)
		max, err = parseNumericStringToPtr(strings.TrimPrefix(num, "<="))
	} else if strings.HasPrefix(num, "<") {
		inclusive = boolPtr(false)
		max, err = parseNumericStringToPtr(strings.TrimPrefix(num, "<"))
	} else if strings.HasPrefix(num, ">=") {
		inclusive = boolPtr(true)
		min, err = parseNumericStringToPtr(strings.TrimPrefix(num, ">="))
	} else if strings.HasPrefix(num, ">") {
		inclusive = boolPtr(false)
		min, err = parseNumericStringToPtr(strings.TrimPrefix(num, ">"))
	} else {
		inclusive = boolPtr(true)
		min, err = parseNumericStringToPtr(num)
		max = min
	}
	return
}

func newNumericQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, val := range values {
		min, max, inclusive, err := parseNumericValue(val)
		if err != nil {
			return nil, err
		}
		q := bleve.NewNumericRangeInclusiveQuery(min, max, inclusive, inclusive)
		q.FieldVal = field
		d.AddQuery(q)
	}
	return d, nil
}

func stringToSeverity(s string) (v1.Severity, error) {
	s = strings.ToLower(s)
	if strings.HasPrefix(s, "l") {
		return v1.Severity_LOW_SEVERITY, nil
	}
	if strings.HasPrefix(s, "m") {
		return v1.Severity_MEDIUM_SEVERITY, nil
	}
	if strings.HasPrefix(s, "h") {
		return v1.Severity_HIGH_SEVERITY, nil
	}
	if strings.HasPrefix(s, "c") {
		return v1.Severity_CRITICAL_SEVERITY, nil
	}
	return v1.Severity_UNSET_SEVERITY, fmt.Errorf("Could not parse severity '%s'. Valid options are low, medium, high, critical", s)
}

func newExactNumericMatch(field string, f float64) query.Query {
	t := true
	q := bleve.NewNumericRangeInclusiveQuery(&f, &f, &t, &t)
	q.FieldVal = field
	return q
}

func newSeverityQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, v := range values {
		sev, err := stringToSeverity(v)
		if err != nil {
			return nil, err
		}
		d.AddQuery(newExactNumericMatch(field, float64(sev)))
	}
	return d, nil
}

func stringToEnforcement(s string) (v1.EnforcementAction, error) {
	s = strings.ToLower(s)
	if strings.Contains(s, "scale") {
		return v1.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, nil
	}
	if strings.Contains(s, "node") {
		return v1.EnforcementAction_UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT, nil
	}
	if strings.Contains(s, "none") {
		return v1.EnforcementAction_UNSET_ENFORCEMENT, nil
	}
	return v1.EnforcementAction_UNSET_ENFORCEMENT, fmt.Errorf("Could not parse enforcement '%s'. Valid options are node, and scale", s)
}

func newEnforcementQuery(field string, values []string) (query.Query, error) {
	d := bleve.NewDisjunctionQuery()
	for _, v := range values {
		en, err := stringToEnforcement(v)
		if err != nil {
			return nil, err
		}
		d.AddQuery(newExactNumericMatch(field, float64(en)))
	}
	return d, nil
}

func fieldsToQuery(fieldMap map[string]*v1.ParsedSearchRequest_Values, objectMap map[string]string) (*query.ConjunctionQuery, error) {
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

func transformFields(fields map[string]*v1.ParsedSearchRequest_Values, objectMap map[string]string) map[string]*v1.ParsedSearchRequest_Values {
	newMap := make(map[string]*v1.ParsedSearchRequest_Values, len(fields))
	for k, v := range fields {
		newMap[transformKey(k, objectMap)] = v
	}
	return newMap
}
