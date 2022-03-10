package pgsearch

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/parse"
	"github.com/stackrox/rox/pkg/postgres/walker"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

type queryFunction func(column string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error)

var datatypeToQueryFunc = map[walker.DataType]queryFunction{
	walker.String:      newStringQuery,
	walker.Bool:        newBoolQuery,
	walker.StringArray: queryOnArray(newStringQuery),
	walker.DateTime:    newTimeQuery,
	walker.Enum:        newEnumQuery,
	walker.Integer:     newNumericQuery,
	walker.Numeric:     newNumericQuery,
	walker.EnumArray:   queryOnArray(newEnumQuery),
	walker.IntArray:    queryOnArray(newNumericQuery),
	// Map is handled separately.

	// TODOs
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

func queryOnArray(baseQuery queryFunction) queryFunction {
	return func(column string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (*QueryEntry, error) {
		baseQ, err := baseQuery("elem", field, value, queryModifiers...)
		if err != nil {
			return nil, err
		}
		baseQ.Query = fmt.Sprintf("exists (select * from unnest(%s) as elem where %s)", column, baseQ.Query)
		return baseQ, nil
	}
}
