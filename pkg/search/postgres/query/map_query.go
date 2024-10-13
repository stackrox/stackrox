package pgsearch

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

// ParseMapQuery parses a label stored in the form k=v.
func ParseMapQuery(label string) (string, string, bool, bool) {
	hasNotEquals := strings.Contains(label, search.NotInMapKeyValueSeparator)
	if hasNotEquals {
		key, value := stringutils.Split2(label, search.NotInMapKeyValueSeparator)
		return key, value, false, hasNotEquals
	}

	hasEquals := strings.Contains(label, search.MapKeyValueSeparator)
	key, value := stringutils.Split2(label, search.MapKeyValueSeparator)
	return key, value, hasEquals, false
}

func readMapValue(val interface{}) map[string]string {
	// Maps are stored in a jsonb column, which we get back as a byte array.
	// We know that supported maps are only map[string]string, so we unmarshal accordingly.
	v, ok := val.(*[]byte)
	if !ok || v == nil || *v == nil {
		return nil
	}
	var mapValue map[string]string
	if err := json.Unmarshal(*v, &mapValue); err != nil {
		utils.Should(err)
		return nil
	}
	return mapValue
}

func newMapQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	// Negations in maps are a bit tricky, as are empty strings. We have to consider the following cases.
	// Note that in everything below, query can be a regex, prefix or exact match query.
	// = => it means we want a non-empty map
	// <keyQuery>= => it means we want at least one element in the map where key matches keyQuery
	// !<keyQuery>= => it means there must no element in the map where key matches keyQuery
	// =<valueQuery> => it means we want one element in the map where value matches valueQuery
	// <keyQuery>=<valueQuery> => straightforward, we want at least one element in the map where key matches keyQuery AND value matches valueQuery
	// !<keyQuery>=<valueQuery> => NOT SUPPORTED
	// =!<valueQuery> => it means we want one element in the map where value does not match valueQuery
	// <keyQuery>=!<valueQuery> => it means we want one element in the map with key matching keyQuery and value NOT matching valueQuery
	// !<keyQuery>=!<valueQuery> => NOT SUPPORTED
	// <keyQuery>!=<valueQuery> => it means there must be no element in the map where key matches keyQuery and value matches valueQuery
	//     <keyQuery>!= , !<keyQuery>!= , !=<valueQuery> , !=!<valueQuery> , != => NOT SUPPORTED
	query := ctx.value
	if query == search.WildcardString {
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query: "true",
		}, func(i interface{}) interface{} {
			asMap := readMapValue(i)
			results := make([]string, 0, len(asMap))
			for k, v := range asMap {
				results = append(results, fmt.Sprintf("%s=%s", k, v))
			}
			return results
		}), nil
	}

	key, value, hasEquals, hasNotEquals := ParseMapQuery(query)
	if hasNotEquals {
		return newNotInMapQuery(ctx, query, key, value)
	}

	keyNegated := stringutils.ConsumePrefix(&key, search.NegationPrefix)
	// This is a special case where the query we construct becomes a (non) existence query
	if value == "" && key != "" && hasEquals {
		var negationString string
		if keyNegated {
			negationString = "NOT "
		}
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query:  fmt.Sprintf("%s(%s ? $$)", negationString, ctx.qualifiedColumnName),
			Values: []interface{}{key},
		}, func(i interface{}) interface{} {
			// If key is negated, no highlight value.
			if keyNegated {
				return []string(nil)
			}
			asMap := readMapValue(i)
			if asMap == nil {
				return []string(nil)
			}
			return []string{fmt.Sprintf("%s=%s", key, asMap[key])}
		}), nil
	}

	if keyNegated {
		return nil, fmt.Errorf("unsupported map query %s: cannot negate key and specify non-empty value", query)
	}

	var keyQuery, valueQuery WhereClause
	if key != "" {
		trimmedKey, keyModifiers := search.GetValueAndModifiersFromString(key)
		var err error
		keyQuery, err = newStringQueryWhereClause("elem.key", trimmedKey, keyModifiers...)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate query for key from %s: %w", query, err)
		}
	}
	if value != "" {
		trimmedValue, valueModifiers := search.GetValueAndModifiersFromString(value)
		var err error
		valueQuery, err = newStringQueryWhereClause("elem.value", trimmedValue, valueModifiers...)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate query for value from %s: %w", query, err)
		}
	}

	combinedWhereClause := &WhereClause{}
	var keyEquivGoFunc, valueEquivGoFunc func(interface{}) bool
	var queryPortion string
	if key == "" && value == "" {
		queryPortion = "true"
	} else if key != "" && value == "" {
		queryPortion = keyQuery.Query
		combinedWhereClause.Values = keyQuery.Values
		keyEquivGoFunc = keyQuery.equivalentGoFunc
	} else if key == "" && value != "" {
		queryPortion = valueQuery.Query
		combinedWhereClause.Values = valueQuery.Values
		valueEquivGoFunc = valueQuery.equivalentGoFunc
	} else {
		queryPortion = fmt.Sprintf("%s and %s", keyQuery.Query, valueQuery.Query)
		combinedWhereClause.Values = append(combinedWhereClause.Values, keyQuery.Values...)
		combinedWhereClause.Values = append(combinedWhereClause.Values, valueQuery.Values...)
		keyEquivGoFunc = keyQuery.equivalentGoFunc
		valueEquivGoFunc = valueQuery.equivalentGoFunc
	}
	combinedWhereClause.Query = fmt.Sprintf("(jsonb_typeof(%s) = 'object') and (exists (select * from jsonb_each_text(%s) elem where %s))", ctx.qualifiedColumnName, ctx.qualifiedColumnName, queryPortion)
	return qeWithSelectFieldIfNeeded(ctx, combinedWhereClause, func(i interface{}) interface{} {
		asMap := readMapValue(i)
		var out []string
		for k, v := range asMap {
			if keyEquivGoFunc != nil && !keyEquivGoFunc(k) {
				continue
			}
			if valueEquivGoFunc != nil && !valueEquivGoFunc(v) {
				continue
			}
			out = append(out, fmt.Sprintf("%s=%s", k, v))
		}
		slices.Sort(out)
		return out
	}), nil
}

func newNotInMapQuery(ctx *queryAndFieldContext, query, key, value string) (*QueryEntry, error) {
	if key == "" || value == "" {
		return nil, fmt.Errorf("unsupported 'not in map' query %s: both key and value cannot be empty", query)
	}
	keyNegated := strings.HasPrefix(key, search.NegationPrefix)
	valueNegated := strings.HasPrefix(value, search.NegationPrefix)
	if keyNegated || valueNegated {
		return nil, fmt.Errorf("unsupported 'not in map' query %s: cannot negate key and value again", query)
	}

	// At this point both key and value are non-empty and not negated
	var keyQuery, valueQuery WhereClause
	var err error

	trimmedKey, keyModifiers := search.GetValueAndModifiersFromString(key)
	keyQuery, err = newStringQueryWhereClause("elem.key", trimmedKey, keyModifiers...)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate query for key from %s: %w", query, err)
	}

	trimmedValue, valueModifiers := search.GetValueAndModifiersFromString(value)
	valueQuery, err = newStringQueryWhereClause("elem.value", trimmedValue, valueModifiers...)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate query for value from %s: %w", query, err)
	}

	var queryPortion string
	var keyEquivGoFunc, valueEquivGoFunc func(interface{}) bool

	combinedWhereClause := &WhereClause{}
	queryPortion = fmt.Sprintf("%s and %s", keyQuery.Query, valueQuery.Query)
	combinedWhereClause.Values = append(combinedWhereClause.Values, keyQuery.Values...)
	combinedWhereClause.Values = append(combinedWhereClause.Values, valueQuery.Values...)
	combinedWhereClause.Query = fmt.Sprintf("(jsonb_typeof(%s) = 'object') and (NOT exists (select * from jsonb_each_text(%s) elem where %s))", ctx.qualifiedColumnName, ctx.qualifiedColumnName, queryPortion)

	keyEquivGoFunc = keyQuery.equivalentGoFunc
	valueEquivGoFunc = valueQuery.equivalentGoFunc
	return qeWithSelectFieldIfNeeded(ctx, combinedWhereClause, func(i interface{}) interface{} {
		asMap := readMapValue(i)
		var out []string
		for k, v := range asMap {
			// keyEquivGoFunc will return true when k matches key in the given query
			// valueEquivGoFunc will return true when v matches value in the given query
			// But this is a non-existence query. So we highlight k, v pairs that do not match either key or value from the given query
			if keyEquivGoFunc != nil && !keyEquivGoFunc(k) {
				out = append(out, fmt.Sprintf("%s=%s", k, v))
				continue
			}
			if valueEquivGoFunc != nil && !valueEquivGoFunc(v) {
				out = append(out, fmt.Sprintf("%s=%s", k, v))
			}
		}
		slices.Sort(out)
		return out
	}), nil
}
