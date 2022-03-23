package pgsearch

import (
	"fmt"

	"github.com/stackrox/rox/pkg/postgres/walker"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

type queryAndFieldContext struct {
	qualifiedColumnName string
	field               *pkgSearch.Field
	dbField             *walker.Field

	value          string
	highlight      bool
	queryModifiers []pkgSearch.QueryModifier
}

func qeWithSelectFieldIfNeeded(ctx *queryAndFieldContext, whereClause *WhereClause, postTransformFunc func(interface{}) interface{}) *QueryEntry {
	qe := &QueryEntry{Where: *whereClause}
	if ctx.highlight {
		qe.SelectedFields = []SelectQueryField{{
			SelectPath:    ctx.qualifiedColumnName,
			FieldType:     ctx.dbField.DataType,
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
	walker.Numeric:     newNumericQuery,
	walker.EnumArray:   queryOnArray(newEnumQuery, getEnumArrayPostTransformFunc),
	walker.IntArray:    queryOnArray(newNumericQuery, getIntArrayPostTransformFunc),
	// Map is handled separately.
}

func matchFieldQuery(dbField *walker.Field, field *pkgSearch.Field, value string, highlight bool) (*QueryEntry, error) {
	qualifiedColName := dbField.Schema.Table + "." + dbField.ColumnName

	ctx := &queryAndFieldContext{
		qualifiedColumnName: qualifiedColName,
		field:               field,
		dbField:             dbField,
		highlight:           highlight,
		value:               value,
	}

	// Special case: wildcard
	if stringutils.MatchesAny(value, pkgSearch.WildcardString, pkgSearch.NullString) {
		return handleExistenceQueries(ctx), nil
	}

	if dbField.DataType == walker.Map {
		return newMapQuery(ctx)
	}

	trimmedValue, modifiers := pkgSearch.GetValueAndModifiersFromString(value)
	ctx.value = trimmedValue
	ctx.queryModifiers = modifiers
	return datatypeToQueryFunc[dbField.DataType](ctx)
}

func handleExistenceQueries(ctx *queryAndFieldContext) *QueryEntry {
	switch ctx.value {
	case pkgSearch.WildcardString:
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query: fmt.Sprintf("%s is not null", ctx.qualifiedColumnName),
		}, nil)
	case pkgSearch.NullString:
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query: fmt.Sprintf("%s is null", ctx.qualifiedColumnName),
		}, nil)
	default:
		log.Fatalf("existence query for value %s is not currently handled", ctx.value)
	}
	return nil
}
