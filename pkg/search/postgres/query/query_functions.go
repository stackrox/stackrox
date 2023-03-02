package pgsearch

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

type queryAndFieldContext struct {
	qualifiedColumnName string
	field               *pkgSearch.Field
	derivedMetadata     *walker.DerivedSearchField
	sqlDataType         postgres.DataType

	value          string
	highlight      bool
	queryModifiers []pkgSearch.QueryModifier

	now time.Time
}

func qeWithSelectFieldIfNeeded(ctx *queryAndFieldContext, whereClause *WhereClause, postTransformFunc func(interface{}) interface{}) *QueryEntry {
	qe := &QueryEntry{Where: *whereClause}

	if ctx.derivedMetadata != nil {
		having := qe.Where
		qe.Having = &having
		qe.Where = WhereClause{}
	}

	if ctx.highlight {
		var cast string
		if ctx.sqlDataType == postgres.UUID {
			cast = "::text"
		}
		qe.SelectedFields = []SelectQueryField{{
			SelectPath: ctx.qualifiedColumnName + cast,
			FieldType:  ctx.sqlDataType,
			FieldPath: func() string {
				if ctx.derivedMetadata == nil {
					return ctx.field.FieldPath
				}
				return "derived." + ctx.derivedMetadata.DerivationType.String() + ctx.field.FieldPath
			}(),
			PostTransform: postTransformFunc,
		}}
	}
	return qe
}

type queryFunction func(ctx *queryAndFieldContext) (*QueryEntry, error)

var datatypeToQueryFunc = map[postgres.DataType]queryFunction{
	postgres.String:      newStringQuery,
	postgres.Bool:        newBoolQuery,
	postgres.StringArray: queryOnArray(newStringQuery, getStringArrayPostTransformFunc),
	postgres.DateTime:    newTimeQuery,
	postgres.Enum:        newEnumQuery,
	postgres.Integer:     newNumericQuery,
	postgres.BigInteger:  newNumericQuery,
	postgres.Numeric:     newNumericQuery,
	postgres.EnumArray:   queryOnArray(newEnumQuery, getEnumArrayPostTransformFunc),
	postgres.IntArray:    queryOnArray(newNumericQuery, getIntArrayPostTransformFunc),
	postgres.UUID:        newUUIDQuery,
	// Map is handled separately.
}

func matchFieldQuery(qualifiedColName string, sqlDataType postgres.DataType, field *pkgSearch.Field, derivedMetadata *walker.DerivedSearchField, value string, highlight bool, now time.Time) (*QueryEntry, error) {
	ctx := &queryAndFieldContext{
		qualifiedColumnName: qualifiedColName,
		field:               field,
		derivedMetadata:     derivedMetadata,
		highlight:           highlight,
		value:               value,
		now:                 now,
		sqlDataType:         sqlDataType,
	}

	if sqlDataType == postgres.Map {
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
		if ctx.derivedMetadata != nil {
			return nil, errors.New("match all (*) and match none (-) are not unsupported values for derived field query")
		}

		if len(qe.SelectedFields) != 1 {
			// If there are no selected fields, then no post transform needs to be used
			return handleExistenceQueries(ctx, nil), nil
		}
		return handleExistenceQueries(ctx, qe.SelectedFields[0].PostTransform), nil
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
