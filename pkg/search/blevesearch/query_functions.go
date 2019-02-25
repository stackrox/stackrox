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

var datatypeToQueryFunc = map[v1.SearchDataType]func(v1.SearchCategory, string, string) (query.Query, error){
	v1.SearchDataType_SEARCH_STRING:   newStringQuery,
	v1.SearchDataType_SEARCH_BOOL:     newBoolQuery,
	v1.SearchDataType_SEARCH_NUMERIC:  newNumericQuery,
	v1.SearchDataType_SEARCH_DATETIME: newTimeQuery,
	v1.SearchDataType_SEARCH_ENUM:     newEnumQuery,
	// Map type is handled specially.
}

func nullQuery(category v1.SearchCategory, path string) query.Query {
	bq := bleve.NewBooleanQuery()
	bq.AddMustNot(getWildcardQuery(path))
	bq.AddMust(typeQuery(category))
	return bq
}

func matchFieldQuery(category v1.SearchCategory, searchFieldPath string, searchFieldType v1.SearchDataType, value string) (query.Query, error) {
	// Special case: wildcard
	if value == pkgSearch.WildcardString {
		return getWildcardQuery(searchFieldPath), nil
	}
	// Special case: null
	if value == pkgSearch.NullString {
		return nullQuery(category, searchFieldPath), nil
	}

	return datatypeToQueryFunc[searchFieldType](category, searchFieldPath, value)
}

func getWildcardQuery(field string) *query.WildcardQuery {
	wq := bleve.NewWildcardQuery("*")
	wq.SetField(field)
	return wq
}

func newBooleanQuery(category v1.SearchCategory) *query.BooleanQuery {
	bq := bleve.NewBooleanQuery()
	// This is where things are interesting. BooleanQuery basically generates a true or false that can be used
	// however we must pipe in the search category because the boolean query returns a true/false designation
	bq.AddMust(typeQuery(category))
	return bq
}

func newStringQuery(category v1.SearchCategory, field string, value string) (query.Query, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("value in search query cannot be empty")
	}
	switch {
	case strings.HasPrefix(value, pkgSearch.NegationPrefix) && len(value) > 1:
		boolQuery := newBooleanQuery(category)
		subQuery, err := newStringQuery(category, field, value[len(pkgSearch.NegationPrefix):])
		if err != nil {
			return nil, fmt.Errorf("error computing sub query under negation: %s %s: %s", field, value, err)
		}
		boolQuery.AddMustNot(subQuery)
		return boolQuery, nil
	case strings.HasPrefix(value, pkgSearch.RegexPrefix) && len(value) > len(pkgSearch.RegexPrefix):
		q := bleve.NewRegexpQuery(value[len(pkgSearch.RegexPrefix):])
		q.SetField(field)
		return q, nil
	case strings.HasPrefix(value, pkgSearch.EqualityPrefixSuffix) && strings.HasSuffix(value, pkgSearch.EqualityPrefixSuffix) && len(value) > 2*len(pkgSearch.EqualityPrefixSuffix):
		q := bleve.NewMatchQuery(value[len(pkgSearch.EqualityPrefixSuffix) : len(value)-len(pkgSearch.EqualityPrefixSuffix)])
		q.SetField(field)
		return q, nil
	default:
		return NewMatchPhrasePrefixQuery(field, value), nil
	}
}

func parseLabel(label string) (string, string) {
	spl := strings.SplitN(label, "=", 2)
	if len(spl) < 2 {
		return spl[0], ""
	}
	return spl[0], spl[1]
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

func newEnumQuery(_ v1.SearchCategory, field, value string) (query.Query, error) {
	enumValues := enums.Get(field, value)
	if len(enumValues) == 0 {
		return nil, fmt.Errorf("could not find corresponding enum at field %q with value %q", field, value)
	}
	dq := bleve.NewDisjunctionQuery()
	for _, s := range enumValues {
		dq.AddQuery(createNumericQuery(field, "=", floatPtr(float64(s))))
	}
	return dq, nil
}

func typeQuery(category v1.SearchCategory) query.Query {
	q := bleve.NewMatchQuery(category.String())
	q.SetField("type")
	return q
}
