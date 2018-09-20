package blevesearch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/stackrox/rox/generated/api/v1"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

const (
	negationPrefix = "!"
	nullString     = "-"
)

var datatypeToQueryFunc = map[v1.SearchDataType]func(v1.SearchCategory, string, string) (query.Query, error){
	v1.SearchDataType_SEARCH_STRING:   newStringQuery,
	v1.SearchDataType_SEARCH_BOOL:     newBoolQuery,
	v1.SearchDataType_SEARCH_NUMERIC:  newNumericQuery,
	v1.SearchDataType_SEARCH_DATETIME: newTimeQuery,

	v1.SearchDataType_SEARCH_SEVERITY:    newSeverityQuery,
	v1.SearchDataType_SEARCH_ENFORCEMENT: newEnforcementQuery,
	v1.SearchDataType_SEARCH_MAP:         newMapQuery,
	v1.SearchDataType_SEARCH_SECRET_TYPE: newSecretTypeQuery,
}

func getWildcardQuery(field string) *query.WildcardQuery {
	wq := bleve.NewWildcardQuery("*")
	wq.SetField(field)
	return wq
}

func matchFieldQuery(category v1.SearchCategory, searchField *v1.SearchField, value string) (query.Query, error) {
	// Special case: null
	if value == nullString {
		bq := bleve.NewBooleanQuery()
		bq.AddMustNot(getWildcardQuery(searchField.GetFieldPath()))
		bq.AddMust(typeQuery(category))
		return bq, nil
	}

	return datatypeToQueryFunc[searchField.GetType()](category, searchField.GetFieldPath(), value)
}

func newStringQuery(category v1.SearchCategory, field string, value string) (query.Query, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("value in search query cannot be empty")
	}
	switch {
	case strings.HasPrefix(value, negationPrefix) && len(value) > 1:
		boolQuery := bleve.NewBooleanQuery()
		boolQuery.AddMustNot(NewMatchPhrasePrefixQuery(field, value[len(negationPrefix):]))
		// This is where things are interesting. BooleanQuery basically generates a true or false that can be used
		// however we must pipe in the search category because the boolean query returns a true/false designation
		boolQuery.AddMust(typeQuery(category))
		return boolQuery, nil
	case strings.HasPrefix(value, pkgSearch.RegexPrefix) && len(value) > len(pkgSearch.RegexPrefix):
		q := bleve.NewRegexpQuery(value[len(pkgSearch.RegexPrefix):])
		q.SetField(field)
		return q, nil
	default:
		return NewMatchPhrasePrefixQuery(field, value), nil
	}
}

func parseLabel(label string) (string, string, error) {
	spl := strings.SplitN(label, "=", 2)
	if len(spl) < 2 {
		return "", "", fmt.Errorf("Malformed label '%s'. Must be in the form key=value", label)
	}
	return spl[0], spl[1], nil
}

func newMapQuery(category v1.SearchCategory, field string, value string) (query.Query, error) {
	mapKey, mapValue, err := parseLabel(value)
	if err != nil {
		return nil, err
	}
	return newStringQuery(category, field+"."+mapKey, mapValue)
}

func newBoolQuery(_ v1.SearchCategory, field string, value string) (query.Query, error) {
	b, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	q := bleve.NewBoolFieldQuery(b)
	q.FieldVal = field
	return q, nil
}

func newSeverityQuery(_ v1.SearchCategory, field string, value string) (query.Query, error) {
	return evaluateEnum(value, field, v1.Severity_name)
}

func newEnforcementQuery(_ v1.SearchCategory, field string, value string) (query.Query, error) {
	return evaluateEnum(value, field, v1.EnforcementAction_name)
}

func newSecretTypeQuery(_ v1.SearchCategory, field string, value string) (query.Query, error) {
	return evaluateEnum(value, field, v1.SecretType_name)
}

func typeQuery(category v1.SearchCategory) query.Query {
	q := bleve.NewMatchQuery(category.String())
	q.SetField("type")
	return q
}
