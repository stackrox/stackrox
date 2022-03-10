package pgsearch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/parse"
	"github.com/stackrox/rox/pkg/postgres/walker"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

type queryFunction func(table string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error)

var datatypeToQueryFunc = map[walker.DataType]queryFunction{
	walker.String:      newStringQuery,
	walker.Bool:        newBoolQuery,
	walker.StringArray: newStringArrayQuery,
	walker.DateTime:    newTimeQuery,
	walker.Enum:        newEnumQuery,
	walker.Integer:     newNumericQuery,
	walker.Numeric:     newNumericQuery,
	// Map is handled separately.

	// TODOs
	// walker.Numeric:
	// walker.Integer:
	// walker.IntArray:
}

func matchFieldQuery(dbField *walker.Field, field *pkgSearch.Field, value string) (*QueryEntry, error) {
	// Special case: wildcard
	if stringutils.MatchesAny(value, pkgSearch.WildcardString, pkgSearch.NullString) {
		return handleExistenceQueries(dbField.ColumnName, value), nil
	}

	if dbField.DataType == walker.Map {
		return newMapQuery(dbField.ColumnName, field, value)
	}

	trimmedValue, modifiers := pkgSearch.GetValueAndModifiersFromString(value)
	return datatypeToQueryFunc[dbField.DataType](dbField.ColumnName, field, trimmedValue, modifiers...)
}

func handleExistenceQueries(root string, value string) *QueryEntry {
	switch value {
	case pkgSearch.WildcardString:
		return &QueryEntry{
			Query: fmt.Sprintf("%s is not null", root),
		}
	case pkgSearch.NullString:
		return &QueryEntry{
			Query: fmt.Sprintf("%s is null", root),
		}
	default:
		log.Fatalf("existence query for value %s is not currently handled", value)
	}
	return nil
}

func newStringArrayQuery(columnName string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	stringQuery, err := newStringQuery("elem", field, value, queryModifiers...)
	if err != nil {
		return nil, err
	}
	// We need to unnest the string array and see if at least one element in the array matches the query
	stringQuery.Query = fmt.Sprintf("exists (select * from unnest(%s) as elem where %s)", columnName, stringQuery.Query)
	return stringQuery, nil
}

func newStringQuery(columnName string, _ *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	if len(value) == 0 {
		return nil, errors.New("value in search query cannot be empty")
	}

	if len(queryModifiers) == 0 {
		return &QueryEntry{
			Query:  fmt.Sprintf("%s ilike $$", columnName),
			Values: []interface{}{value + "%"},
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

	if len(queryModifiers) == 0 {
		return &QueryEntry{
			Query:  fmt.Sprintf("NOT (%s ilike $$)", columnName),
			Values: []interface{}{value + "%"},
		}, nil
	}

	switch queryModifiers[0] {
	case pkgSearch.Regex:
		return &QueryEntry{
			Query:  fmt.Sprintf("%s %s~* $$", columnName, negationString),
			Values: []interface{}{value},
		}, nil
	case pkgSearch.Equality:
		return &QueryEntry{
			Query:  fmt.Sprintf("%s %s= $$", columnName, negationString),
			Values: []interface{}{value},
		}, nil
	}
	err := errors.Errorf("unknown query modifier: %s", queryModifiers[0])
	utils.Should(err)
	return nil, err
}

func newBoolQuery(column string, field *pkgSearch.Field, value string, modifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	if len(modifiers) > 0 {
		return nil, errors.Errorf("modifiers for bool query not allowed: %+v", modifiers)
	}
	res, err := parse.FriendlyParseBool(value)
	if err != nil {
		return nil, err
	}
	// explicitly apply equality check
	return newStringQuery(column, field, strconv.FormatBool(res), pkgSearch.Equality)
}

func enumEquality(columnName string, field *pkgSearch.Field, enumValues []int32) (*QueryEntry, error) {
	var queries []string
	var values []interface{}
	for _, s := range enumValues {
		entry, err := newStringQuery(columnName, field, strconv.Itoa(int(s)), pkgSearch.Equality)
		if err != nil {
			return nil, err
		}
		queries = append(queries, entry.Query)
		values = append(values, entry.Values...)
	}
	return &QueryEntry{
		Query:  fmt.Sprintf("(%s)", strings.Join(queries, " or ")),
		Values: values,
	}, nil
}

func newEnumQuery(columnName string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
	var enumValues []int32
	if len(queryModifiers) > 2 {
		return nil, errors.Errorf("unsupported: more than 2 query modifiers for enum query: %+v", queryModifiers)
	}
	var equality bool
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
			equality = true
		}
		enumValues = enumregistry.Get(field.FieldPath, value)
		if len(enumValues) == 0 {
			return NewFalseQuery(), nil
		}

		// Equality means no numeric cast required, and could benefit from hash indexes
		if equality {
			return enumEquality(columnName, field, enumValues)
		}

		var queries []string
		var values []interface{}
		for _, s := range enumValues {
			entry := createNumericQuery(columnName, field, prefix, float64(s))
			queries = append(queries, entry.Query)
			values = append(values, entry.Values...)
		}
		return &QueryEntry{
			Query:  fmt.Sprintf("(%s)", strings.Join(queries, " or ")),
			Values: values,
		}, nil
	}

	if len(enumValues) == 0 {
		return nil, fmt.Errorf("could not find corresponding enum at field %q with value %q and modifiers %+v", field.FieldPath, value, queryModifiers)
	}
	return enumEquality(columnName, field, enumValues)
}
