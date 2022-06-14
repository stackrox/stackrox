package blevesearch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/parse"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/enumregistry"
	"github.com/stackrox/stackrox/pkg/utils"
)

type queryFunction func(category v1.SearchCategory, field, value string, queryModifiers ...queryModifier) (query.Query, error)

var datatypeToQueryFunc = map[v1.SearchDataType]queryFunction{
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

//go:generate stringer -type=queryModifier
type queryModifier int

const (
	atLeastOne queryModifier = iota
	negation
	regex
	equality
)

func matchFieldQuery(category v1.SearchCategory, searchFieldPath string, searchFieldType v1.SearchDataType, value string) (query.Query, error) {
	// Special case: wildcard
	if value == pkgSearch.WildcardString {
		return getWildcardQuery(searchFieldPath), nil
	}
	// Special case: null
	if value == pkgSearch.NullString {
		return nullQuery(category, searchFieldPath), nil
	}

	// Parse out query modifiers
	trimmedValue := value
	var queryModifiers []queryModifier
	// We only allow at most one modifier from the set {atleastone, negation}.
	// Anything more, we treat as part of the string to query for.
	var negationOrAtLeastOneFound bool
forloop:
	for {
		switch {
		// AtLeastOnePrefix is !! so it must come before negation prefix
		case !negationOrAtLeastOneFound && strings.HasPrefix(trimmedValue, pkgSearch.AtLeastOnePrefix) && len(trimmedValue) > len(pkgSearch.AtLeastOnePrefix):
			trimmedValue = trimmedValue[len(pkgSearch.AtLeastOnePrefix):]
			queryModifiers = append(queryModifiers, atLeastOne)
			negationOrAtLeastOneFound = true
		case !negationOrAtLeastOneFound && strings.HasPrefix(trimmedValue, pkgSearch.NegationPrefix) && len(trimmedValue) > len(pkgSearch.NegationPrefix):
			trimmedValue = trimmedValue[len(pkgSearch.NegationPrefix):]
			queryModifiers = append(queryModifiers, negation)
			negationOrAtLeastOneFound = true
		case strings.HasPrefix(trimmedValue, pkgSearch.RegexPrefix) && len(trimmedValue) > len(pkgSearch.RegexPrefix):
			trimmedValue = strings.ToLower(trimmedValue[len(pkgSearch.RegexPrefix):])
			queryModifiers = append(queryModifiers, regex)
			break forloop // Once we see that it's a regex, we don't check for special-characters in the rest of the string.
		case strings.HasPrefix(trimmedValue, pkgSearch.EqualityPrefixSuffix) && strings.HasSuffix(trimmedValue, pkgSearch.EqualityPrefixSuffix) && len(trimmedValue) > 2*len(pkgSearch.EqualityPrefixSuffix):
			trimmedValue = trimmedValue[len(pkgSearch.EqualityPrefixSuffix) : len(trimmedValue)-len(pkgSearch.EqualityPrefixSuffix)]
			queryModifiers = append(queryModifiers, equality)
			break forloop // Once it's within quotes, we take the value inside as is, and don't try to extract modifiers.
		default:
			break forloop
		}
	}

	return datatypeToQueryFunc[searchFieldType](category, searchFieldPath, trimmedValue, queryModifiers...)
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

func newStringQuery(category v1.SearchCategory, field string, value string, queryModifiers ...queryModifier) (query.Query, error) {
	if len(value) == 0 {
		return nil, errors.New("value in search query cannot be empty")
	}
	if len(queryModifiers) == 0 {
		return NewMatchPhrasePrefixQuery(field, value), nil
	}
	switch queryModifiers[0] {
	case atLeastOne:
		subQuery, err := newStringQuery(category, field, value, queryModifiers[1:]...)
		if err != nil {
			return nil, errors.Wrapf(err, "error computing sub query under negation: %s %s", field, value)
		}
		nq := NewNegationQuery(typeQuery(category), subQuery, true)
		return nq, nil
	case negation:
		subQuery, err := newStringQuery(category, field, value, queryModifiers[1:]...)
		if err != nil {
			return nil, errors.Wrapf(err, "error computing sub query under negation: %s %s", field, value)
		}
		bq := newBooleanQuery(category)
		bq.AddMustNot(subQuery)
		return bq, nil
	case regex:
		q := bleve.NewRegexpQuery(value)
		q.SetField(field)
		return q, nil
	case equality:
		q := bleve.NewMatchQuery(value)
		q.SetField(field)
		return q, nil
	}
	err := errors.Errorf("unknown query modifier: %s", queryModifiers[0])
	utils.Should(err)
	return nil, err
}

func parseLabel(label string) (string, string) {
	spl := strings.SplitN(label, "=", 2)
	if len(spl) < 2 {
		return spl[0], ""
	}
	return spl[0], spl[1]
}

func newBoolQuery(_ v1.SearchCategory, field string, value string, modifiers ...queryModifier) (query.Query, error) {
	if len(modifiers) > 0 {
		return nil, errors.Errorf("modifiers for bool query not allowed: %+v", modifiers)
	}
	b, err := parse.FriendlyParseBool(value)
	if err != nil {
		return nil, err
	}
	q := bleve.NewBoolFieldQuery(b)
	q.FieldVal = field
	return q, nil
}

func newEnumQuery(_ v1.SearchCategory, field, value string, queryModifiers ...queryModifier) (query.Query, error) {
	var enumValues []int32
	if len(queryModifiers) > 2 {
		return nil, errors.Errorf("unsupported: more than 2 query modifiers for enum query: %+v", queryModifiers)
	}
	switch len(queryModifiers) {
	case 2:
		if queryModifiers[0] == negation && queryModifiers[1] == regex {
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid regex")
			}

			enumValues = enumregistry.GetComplementOfValuesMatchingRegex(field, re)
			break
		}
		if queryModifiers[0] == negation && queryModifiers[1] == equality {
			enumValues = enumregistry.GetComplementByExactMatches(field, value)
			break
		}
		return nil, errors.Errorf("unsupported: invalid combination of query modifiers for enum query: %+v", queryModifiers)
	case 1:
		switch queryModifiers[0] {
		case negation:
			enumValues = enumregistry.GetComplement(field, value)
		case regex:
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid regex")
			}
			enumValues = enumregistry.GetValuesMatchingRegex(field, re)
		case equality:
			enumValues = enumregistry.GetExactMatches(field, value)
		default:
			return nil, errors.Errorf("unsupported query modifier for enum query: %v", queryModifiers[0])
		}
	case 0:
		prefix, value := parseNumericPrefix(value)
		if prefix == "" {
			prefix = "="
		}
		enumValues = enumregistry.Get(field, value)
		dq := bleve.NewDisjunctionQuery()
		for _, s := range enumValues {
			dq.AddQuery(createNumericQuery(field, prefix, floatPtr(float64(s))))
		}
		return dq, nil
	}

	if len(enumValues) == 0 {
		return nil, fmt.Errorf("could not find corresponding enum at field %q with value %q and modifiers %+v", field, value, queryModifiers)
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
