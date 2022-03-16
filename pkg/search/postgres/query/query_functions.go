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

type queryAndFieldContext struct {
	qualifiedColumnName string
	field               *pkgSearch.Field
	dbField             *walker.Field

	value          string
	highlight      bool
	queryModifiers []pkgSearch.QueryModifier
}

type queryFunction func(ctx *queryAndFieldContext) (*QueryEntry, error)

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
}

func matchFieldQuery(dbField *walker.Field, field *pkgSearch.Field, value string, highlight bool) (*QueryEntry, error) {
	qualifiedColName := dbField.Schema.Table + "." + dbField.ColumnName
	// Special case: wildcard
	if stringutils.MatchesAny(value, pkgSearch.WildcardString, pkgSearch.NullString) {
		return handleExistenceQueries(qualifiedColName, value), nil
	}

	if dbField.DataType == walker.Map {
		return newMapQuery(qualifiedColName, value, highlight)
	}

	trimmedValue, modifiers := pkgSearch.GetValueAndModifiersFromString(value)
	return datatypeToQueryFunc[dbField.DataType](&queryAndFieldContext{
		qualifiedColumnName: qualifiedColName,
		field:               field,
		dbField:             dbField,
		value:               trimmedValue,
		highlight:           highlight,
		queryModifiers:      modifiers,
	})
}

func handleExistenceQueries(root string, value string) *QueryEntry {
	switch value {
	case pkgSearch.WildcardString:
		return &QueryEntry{Where: WhereClause{
			Query: fmt.Sprintf("%s is not null", root),
		}}
	case pkgSearch.NullString:
		return &QueryEntry{Where: WhereClause{
			Query: fmt.Sprintf("%s is null", root),
		}}
	default:
		log.Fatalf("existence query for value %s is not currently handled", value)
	}
	return nil
}

func newStringQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	whereClause, err := newStringQueryWhereClause(ctx.qualifiedColumnName, ctx.value, ctx.queryModifiers...)
	if err != nil {
		return nil, err
	}
	qe := &QueryEntry{Where: whereClause}
	if ctx.highlight {
		qe.SelectedFields = []SelectQueryField{{SelectPath: ctx.qualifiedColumnName, FieldPath: ctx.field.FieldPath, FieldType: ctx.dbField.DataType}}
	}
	return qe, nil
}

func newStringQueryWhereClause(columnName string, value string, queryModifiers ...pkgSearch.QueryModifier) (WhereClause, error) {
	if len(value) == 0 {
		return WhereClause{}, errors.New("value in search query cannot be empty")
	}

	if len(queryModifiers) == 0 {
		return WhereClause{
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
		return WhereClause{
			Query:  fmt.Sprintf("NOT (%s ilike $$)", columnName),
			Values: []interface{}{value + "%"},
		}, nil
	}

	switch queryModifiers[0] {
	case pkgSearch.Regex:
		return WhereClause{
			Query:  fmt.Sprintf("%s %s~* $$", columnName, negationString),
			Values: []interface{}{value},
		}, nil
	case pkgSearch.Equality:
		return WhereClause{
			Query:  fmt.Sprintf("%s %s= $$", columnName, negationString),
			Values: []interface{}{value},
		}, nil
	}
	err := errors.Errorf("unknown query modifier: %s", queryModifiers[0])
	utils.Should(err)
	return WhereClause{}, err
}

func newBoolQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	if len(ctx.queryModifiers) > 0 {
		return nil, errors.Errorf("modifiers for bool query not allowed: %+v", ctx.queryModifiers)
	}
	res, err := parse.FriendlyParseBool(ctx.value)
	if err != nil {
		return nil, err
	}
	// explicitly apply equality check
	ctx.value = strconv.FormatBool(res)
	ctx.queryModifiers = []pkgSearch.QueryModifier{pkgSearch.Equality}
	return newStringQuery(ctx)
}

func queryOnArray(baseQuery queryFunction) queryFunction {
	return func(ctx *queryAndFieldContext) (*QueryEntry, error) {
		clonedCtx := *ctx
		clonedCtx.highlight = false
		clonedCtx.qualifiedColumnName = "elem"
		baseQ, err := baseQuery(&clonedCtx)
		if err != nil {
			return nil, err
		}
		// If no highlight, use an exists query since SQL optimizes that (by early exiting)
		if !ctx.highlight {
			baseQ.Where.Query = fmt.Sprintf("exists (select * from unnest(%s) as elem where %s)", ctx.qualifiedColumnName, baseQ.Where.Query)
			return &QueryEntry{Where: baseQ.Where}, nil
		}
		return nil, errors.New("highlights not supported yet")
	}
}
