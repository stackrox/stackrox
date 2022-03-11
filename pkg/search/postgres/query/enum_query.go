package pgsearch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/enumregistry"
)

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
		return nil, fmt.Errorf("unsupported: more than 2 query modifiers for enum query: %+v", queryModifiers)
	}
	var equality bool
	switch len(queryModifiers) {
	case 2:
		if queryModifiers[0] == pkgSearch.Negation && queryModifiers[1] == pkgSearch.Regex {
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("invalid regex: %w", err)
			}

			enumValues = enumregistry.GetComplementOfValuesMatchingRegex(field.FieldPath, re)
			break
		}
		if queryModifiers[0] == pkgSearch.Negation && queryModifiers[1] == pkgSearch.Equality {
			enumValues = enumregistry.GetComplementByExactMatches(field.FieldPath, value)
			break
		}
		return nil, fmt.Errorf("unsupported: invalid combination of query modifiers for enum query: %+v", queryModifiers)
	case 1:
		switch queryModifiers[0] {
		case pkgSearch.Negation:
			enumValues = enumregistry.GetComplement(field.FieldPath, value)
		case pkgSearch.Regex:
			re, err := regexp.Compile(value)
			if err != nil {
				return nil, fmt.Errorf("invalid regex: %w", err)
			}
			enumValues = enumregistry.GetValuesMatchingRegex(field.FieldPath, re)
		case pkgSearch.Equality:
			enumValues = enumregistry.GetExactMatches(field.FieldPath, value)
		default:
			return nil, fmt.Errorf("unsupported query modifier for enum query: %v", queryModifiers[0])
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
