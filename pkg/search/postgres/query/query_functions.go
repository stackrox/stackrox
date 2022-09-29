package pgsearch

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/postgres/walker"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

type queryAndFieldContext struct {
	qualifiedColumnName string
	field               *pkgSearch.Field
	sqlDataType         walker.DataType

	value          string
	highlight      bool
	queryModifiers []pkgSearch.QueryModifier

	now time.Time
}

func qeWithSelectFieldIfNeeded(ctx *queryAndFieldContext, whereClause *WhereClause, postTransformFunc func(interface{}) interface{}) *QueryEntry {
	qe := &QueryEntry{Where: *whereClause}
	if ctx.highlight {
		qe.SelectedFields = []SelectQueryField{{
			SelectPath:    ctx.qualifiedColumnName,
			FieldType:     ctx.sqlDataType,
			FieldPath:     ctx.field.FieldPath,
			PostTransform: postTransformFunc,
		}}
	}
	return qe
}

type queryFunction func(ctx *queryAndFieldContext) (*QueryEntry, error)

var datatypeToQueryFunc = map[walker.DataType]queryFunction{
	walker.String:      newStringQuery,
	walker.Bool:        newBoolQuery,
	walker.StringArray: queryOnArray(newStringQuery, getStringArrayPostTransformFunc),
	walker.DateTime:    newTimeQuery,
	walker.Enum:        newEnumQuery,
	walker.Integer:     newNumericQuery,
	walker.BigInteger:  newNumericQuery,
	walker.Numeric:     newNumericQuery,
	walker.EnumArray:   queryOnArray(newEnumQuery, getEnumArrayPostTransformFunc),
	walker.IntArray:    queryOnArray(newNumericQuery, getIntArrayPostTransformFunc),
	// Map is handled separately.
}

func matchFieldQuery(qualifiedColName string, sqlDataType walker.DataType, field *pkgSearch.Field, value string, highlight bool, now time.Time, goesIntoHavingClause bool) (*QueryEntry, error) {
	ctx := &queryAndFieldContext{
		qualifiedColumnName: qualifiedColName,
		field:               field,
		highlight:           highlight,
		value:               value,
		now:                 now,
		sqlDataType:         sqlDataType,
	}

	if sqlDataType == walker.Map {
		return newMapQuery(ctx)
	}

	trimmedValue, modifiers := pkgSearch.GetValueAndModifiersFromString(value)
	ctx.value = trimmedValue
	ctx.queryModifiers = modifiers

	qe, err := datatypeToQueryFunc[sqlDataType](ctx)
	if err != nil {
		return nil, err
	}

	// Special case: wildcard
	if stringutils.MatchesAny(value, pkgSearch.WildcardString, pkgSearch.NullString) {
		if len(qe.SelectedFields) != 1 {
			// If there are no selected fields, then no post transform needs to be used
			return handleExistenceQueries(ctx, nil), nil
		}
		return handleExistenceQueries(ctx, qe.SelectedFields[0].PostTransform), nil
	}

	if goesIntoHavingClause {
		having := qe.Where
		qe.Having = &having
		qe.Where = WhereClause{}
	}
	return qe, nil
}

func handleExistenceQueries(ctx *queryAndFieldContext, transformFn func(interface{}) interface{}) *QueryEntry {
	switch ctx.value {
	case pkgSearch.WildcardString:
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query: fmt.Sprintf("%s is not null", ctx.qualifiedColumnName),
		}, transformFn)
	case pkgSearch.NullString:
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query: fmt.Sprintf("%s is null", ctx.qualifiedColumnName),
		}, transformFn)
	default:
		log.Fatalf("existence query for value %s is not currently handled", ctx.value)
	}
	return nil
}
