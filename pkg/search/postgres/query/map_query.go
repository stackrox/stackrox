package pgsearch

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

func parseMapQuery(label string) (string, string) {
	spl := strings.SplitN(label, "=", 2)
	if len(spl) < 2 {
		return spl[0], ""
	}
	return spl[0], spl[1]
}

func newMapQuery(column string, field *search.Field, query string) (*QueryEntry, error) {
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

	key, value := parseMapQuery(query)

	keyNegated := stringutils.ConsumePrefix(&key, search.NegationPrefix)
	// This is a special case where the query we construct becomes a (non) existence query
	if value == "" && key != "" {
		var negationString string
		if keyNegated {
			negationString = "NOT "
		}
		return &QueryEntry{Query: fmt.Sprintf("%s(%s ? $$)", negationString, column), Values: []interface{}{key}}, nil
	}
	if keyNegated {
		return nil, fmt.Errorf("unsupported map query %s: cannot negate key and specify non-empty value", query)
	}

	var keyQuery, valueQuery *QueryEntry
	if key != "" {
		trimmedKey, keyModifiers := search.GetValueAndModifiersFromString(key)
		var err error
		keyQuery, err = newStringQuery("elem.key", field, trimmedKey, keyModifiers...)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate query for key from %s: %w", query, err)
		}
	}
	if value != "" {
		trimmedValue, valueModifiers := search.GetValueAndModifiersFromString(value)
		var err error
		valueQuery, err = newStringQuery("elem.value", field, trimmedValue, valueModifiers...)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate query for value from %s: %w", query, err)
		}
	}

	combinedQuery := &QueryEntry{}
	var queryPortion string
	if keyQuery == nil && valueQuery == nil {
		queryPortion = "true"
	} else if keyQuery != nil && valueQuery == nil {
		queryPortion = keyQuery.Query
		combinedQuery.Values = keyQuery.Values
	} else if keyQuery == nil && valueQuery != nil {
		queryPortion = valueQuery.Query
		combinedQuery.Values = valueQuery.Values
	} else {
		queryPortion = fmt.Sprintf("%s and %s", keyQuery.Query, valueQuery.Query)
		combinedQuery.Values = append(combinedQuery.Values, keyQuery.Values...)
		combinedQuery.Values = append(combinedQuery.Values, valueQuery.Values...)
	}
	combinedQuery.Query = fmt.Sprintf("exists (select * from jsonb_each_text(%s) elem where %s)", column, queryPortion)
	return combinedQuery, nil
}
