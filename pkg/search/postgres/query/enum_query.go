package pgsearch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
)

func enumEquality(columnName string, enumValues []int32) (WhereClause, error) {
	var queries []string
	var values []interface{}
	for _, s := range enumValues {
		entry, err := newStringQueryWhereClause(columnName, strconv.Itoa(int(s)), pkgSearch.Equality)
		if err != nil {
			return WhereClause{}, err
		}
		queries = append(queries, entry.Query)
		values = append(values, entry.Values...)
	}
	return WhereClause{
		Query:  fmt.Sprintf("(%s)", strings.Join(queries, " or ")),
		Values: values,
		equivalentGoFunc: func(foundValue interface{}) bool {
			asInt := int32(foundValue.(int))
			for _, enumValue := range enumValues {
				if enumValue == asInt {
					return true
				}
			}
			return false
		},
	}, nil
}

func newEnumQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	whereClause, err := newEnumQueryWhereClause(ctx.qualifiedColumnName, ctx.field, ctx.value, ctx.queryModifiers...)
	if err != nil {
		return nil, err
	}
	qe := qeWithSelectFieldIfNeeded(ctx, &whereClause, func(i interface{}) interface{} {
		return enumregistry.Lookup(ctx.field.FieldPath, int32(*(i.(*int))))
	})
	qe.enumStringifyFunc = func(i int32) string {
		return enumregistry.Lookup(ctx.field.FieldPath, i)
	}
	return qe, nil
}

func newEnumQueryWhereClause(columnName string, field *pkgSearch.Field, value string, queryModifiers ...pkgSearch.QueryModifier) (WhereClause, error) {
	var enumValues []int32
	if len(queryModifiers) > 2 {
		return WhereClause{}, fmt.Errorf("unsupported: more than 2 query modifiers for enum query: %+v", queryModifiers)
	}
	var equality bool
	switch len(queryModifiers) {
	case 2:
		if queryModifiers[0] == pkgSearch.Negation && queryModifiers[1] == pkgSearch.Regex {
			re, err := regexp.Compile(value)
			if err != nil {
				return WhereClause{}, fmt.Errorf("invalid regex: %w", err)
			}

			enumValues = enumregistry.GetComplementOfValuesMatchingRegex(field.FieldPath, re)
			break
		}
		if queryModifiers[0] == pkgSearch.Negation && queryModifiers[1] == pkgSearch.Equality {
			enumValues = enumregistry.GetComplementByExactMatches(field.FieldPath, value)
			break
		}
		return WhereClause{}, fmt.Errorf("unsupported: invalid combination of query modifiers for enum query: %+v", queryModifiers)
	case 1:
		switch queryModifiers[0] {
		case pkgSearch.Negation:
			enumValues = enumregistry.GetComplement(field.FieldPath, value)
		case pkgSearch.Regex:
			re, err := regexp.Compile(value)
			if err != nil {
				return WhereClause{}, fmt.Errorf("invalid regex: %w", err)
			}
			enumValues = enumregistry.GetValuesMatchingRegex(field.FieldPath, re)
		case pkgSearch.Equality:
			enumValues = enumregistry.GetExactMatches(field.FieldPath, value)
		default:
			return WhereClause{}, fmt.Errorf("unsupported query modifier for enum query: %v", queryModifiers[0])
		}
	case 0:
		prefix, value := parseNumericPrefix(value)
		if prefix == "" {
			equality = true
		}
		enumValues = enumregistry.Get(field.FieldPath, value)
		if len(enumValues) == 0 {
			return NewFalseQuery().Where, nil
		}

		// Equality means no numeric cast required, and could benefit from hash indexes
		if equality {
			return enumEquality(columnName, enumValues)
		}

		var queries []string
		var values []interface{}
		var equivalentGoFuncs []func(interface{}) bool
		for _, s := range enumValues {
			entry := createNumericPrefixQuery(columnName, prefix, float64(s))
			queries = append(queries, entry.Query)
			values = append(values, entry.Values...)
			equivalentGoFuncs = append(equivalentGoFuncs, entry.equivalentGoFunc)
		}
		return WhereClause{
			Query:  fmt.Sprintf("(%s)", strings.Join(queries, " or ")),
			Values: values,
			equivalentGoFunc: func(foundValue interface{}) bool {
				for _, f := range equivalentGoFuncs {
					if f(foundValue) {
						return true
					}
				}
				return false
			},
		}, nil
	}

	if len(enumValues) == 0 {
		return WhereClause{}, fmt.Errorf("could not find corresponding enum at field %q with value %q and modifiers %+v", field.FieldPath, value, queryModifiers)
	}
	return enumEquality(columnName, enumValues)
}
