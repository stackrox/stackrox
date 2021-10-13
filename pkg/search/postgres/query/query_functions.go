package pgsearch

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/parse"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"

	//"github.com/stackrox/rox/pkg/search/enumregistry"
	"github.com/stackrox/rox/pkg/utils"
)

type queryFunction func(table string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error)

var datatypeToQueryFunc = map[v1.SearchDataType]queryFunction{
	v1.SearchDataType_SEARCH_STRING:   newStringQuery,
	v1.SearchDataType_SEARCH_BOOL:     newBoolQuery,
	v1.SearchDataType_SEARCH_NUMERIC:  newNumericQuery,
	v1.SearchDataType_SEARCH_DATETIME: newTimeQuery,
	v1.SearchDataType_SEARCH_ENUM:     newEnumQuery,
	// Map type is handled specially.
}

//func nullQuery(category v1.SearchCategory, path string) query.Query {
//	bq := bleve.NewBooleanQuery()
//	bq.AddMustNot(getWildcardQuery(path))
//	bq.AddMust(typeQuery(category))
//	return bq
//}

func matchFieldQuery(table string, field *pkgSearch.Field, value string) (*QueryEntry, error) {
	// Special case: wildcard
	if value == pkgSearch.WildcardString {
		log.Infof("Wildcard for %s", field.FieldPath)
		return nil, nil
		//panic("wildcard")
	}
	// Special case: null
	if value == pkgSearch.NullString {
		panic("null string")
	}

	trimmedValue, modifiers := pkgSearch.GetValueAndModifiersFromString(value)
	return datatypeToQueryFunc[field.GetType()](table, field, trimmedValue, modifiers...)
}

//func getWildcardQuery(field string) *query.WildcardQuery {
//	wq := bleve.NewWildcardQuery("*")
//	wq.SetField(field)
//	return wq
//}

func renderFinalPath(elemPath string, field string) string {
	if elemPath == "" {
		return field + " "
	}
	return fmt.Sprintf("%s ->>'%s' ", elemPath, field)
}

func newStringQuery(table string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	if len(value) == 0 {
		return nil, errors.New("value in search query cannot be empty")
	}

	lastElem := field.LastElem()
	//if lastElem.Slice {
	//	panic("need to fix the array attribution to expand this")
	//}
	elemPath := generateShortestElemPath(table, field.Elems)
	if len(queryModifiers) == 0 {
		return &QueryEntry{
			Query: renderFinalPath(elemPath, lastElem.Name) + "ilike $$",
			Values: []interface{}{value+"%"},
		}, nil
	}
	if queryModifiers[0] == pkgSearch.AtLeastOne {
		panic("I dont think this is used")
	}
	var negationString string
	if negated := queryModifiers[0] == pkgSearch.Negation; negated {
		negationString = "!"
		queryModifiers = queryModifiers[1:]
	}

	switch queryModifiers[0] {
	case pkgSearch.Regex:
		return &QueryEntry{
			Query:  renderFinalPath(elemPath, lastElem.Name) + fmt.Sprintf("%s~* $$", negationString),
			Values: []interface{}{value},
		}, nil
	case pkgSearch.Equality:
		return &QueryEntry{
			Query:   renderFinalPath(elemPath, lastElem.Name) + fmt.Sprintf("%s= $$", negationString),
			Values: []interface{}{value},
		}, nil
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

func newBoolQuery(table string, field *pkgSearch.Field, value string, modifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	if len(modifiers) > 0 {
		return nil, errors.Errorf("modifiers for bool query not allowed: %+v", modifiers)
	}
	_, err := parse.FriendlyParseBool(value)
	if err != nil {
		return nil, err
	}
	return newStringQuery(table, field, value, modifiers...)
}

func newEnumQuery(table string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	var enumValues []int32
	if len(queryModifiers) > 2 {
		return nil, errors.Errorf("unsupported: more than 2 query modifiers for enum query: %+v", queryModifiers)
	}
	switch len(queryModifiers) {
	case 2:
		if queryModifiers[0] == pkgSearch.Negation && queryModifiers[1] == pkgSearch.Regex {
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid regex")
			}

			enumValues = enumregistry.GetComplementOfValuesMatchingRegex(field.FieldPath, re)
			break
		}
		if queryModifiers[0] == pkgSearch.Negation && queryModifiers[1] == pkgSearch.Equality {
			enumValues = enumregistry.GetComplementByExactMatches(field.FieldPath, value)
			break
		}
		return nil, errors.Errorf("unsupported: invalid combination of query modifiers for enum query: %+v", queryModifiers)
	case 1:
		switch queryModifiers[0] {
		case pkgSearch.Negation:
			enumValues = enumregistry.GetComplement(field.FieldPath, value)
		case pkgSearch.Regex:
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, errors.Wrap(err, "invalid regex")
			}
			enumValues = enumregistry.GetValuesMatchingRegex(field.FieldPath, re)
		case pkgSearch.Equality:
			enumValues = enumregistry.GetExactMatches(field.FieldPath, value)
		default:
			return nil, errors.Errorf("unsupported query modifier for enum query: %v", queryModifiers[0])
		}
	case 0:
		prefix, value := parseNumericPrefix(value)
		if prefix == "" {
			prefix = "="
		}
		enumValues = enumregistry.Get(field.FieldPath, value)
		if len(enumValues) == 0 {
			return NewFalseQuery(), nil
		}

		var queries []string
		var values []interface{}
		for _, s := range enumValues {
			entry := createNumericQuery(table, field, prefix, floatPtr(float64(s)))
			queries = append(queries, entry.Query)
			values = append(values, entry.Values...)
		}
		return &QueryEntry{
			Query:  fmt.Sprintf("(%s)", strings.Join(queries, " or ")),
			Values: values,
		}, nil
	}

	if len(enumValues) == 0 {
		return nil, fmt.Errorf("could not find corresponding enum at field %q with value %q and modifiers %+v", field, value, queryModifiers)
	}

	var queries []string
	var values []interface{}
	for _, s := range enumValues {
		entry := createNumericQuery(table, field, "=", floatPtr(float64(s)))
		queries = append(queries, entry.Query)
		values = append(values, entry.Values...)
	}
	return &QueryEntry{
		Query:  fmt.Sprintf("(%s)", strings.Join(queries, " or ")),
		Values: values,
	}, nil
}
