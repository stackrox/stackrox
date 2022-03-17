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
	walker.EnumArray:   queryOnArray(newEnumQuery, nil),
	walker.IntArray:    queryOnArray(newNumericQuery, getIntArrayPostTransformFunc),
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
