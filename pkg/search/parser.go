package search

import (
	"errors"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// ParseRawQuery takes the text based query and converts to the query proto.
// It expects the received query to be non-empty.
func ParseRawQuery(query string) (*v1.Query, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	pairs := strings.Split(query, "+")

	queries := make([]*v1.Query, 0, len(pairs))
	for _, pair := range pairs {
		key, commaSeparatedValues, valid := parsePair(pair)
		if !valid {
			continue
		}
		queries = append(queries, queryFromKeyValue(key, commaSeparatedValues))
	}

	if len(queries) == 0 {
		return nil, errors.New("after parsing, query is empty")
	}

	return ConjunctionQuery(queries...), nil
}

func queryFromKeyValue(key, commaSeparatedValues string) *v1.Query {
	// Check if it's a raw string query.
	if strings.EqualFold(key, "has") {
		return stringQuery(commaSeparatedValues)
	}

	valueSlice := strings.Split(commaSeparatedValues, ",")

	return queryFromFieldValues(key, valueSlice)
}

// Extracts "key", "value1,value2" from a string in the format key:value1,value2
func parsePair(pair string) (key string, values string, valid bool) {
	pair = strings.TrimSpace(pair)
	if len(pair) == 0 {
		return
	}

	spl := strings.SplitN(pair, ":", 2)
	// len < 2 implies there isn't a colon and the second check verifies that the : wasn't the last char
	if len(spl) < 2 || spl[1] == "" {
		return
	}
	return spl[0], spl[1], true
}

func queryFromFieldValues(field string, values []string) *v1.Query {
	queries := make([]*v1.Query, 0, len(values))
	for _, value := range values {
		queries = append(queries, matchFieldQuery(field, value))
	}

	return DisjunctionQuery(queries...)
}

// DisjunctionQuery returns a disjunction query of the provided queries.
func DisjunctionQuery(queries ...*v1.Query) *v1.Query {
	return disjunctOrConjunctQueries(false, queries...)
}

// ConjunctionQuery returns a conjunction query of the provided queries.
func ConjunctionQuery(queries ...*v1.Query) *v1.Query {
	return disjunctOrConjunctQueries(true, queries...)
}

// Helper function that DisjunctionQuery and ConjunctionQuery proxy to.
// Do NOT call this directly.
func disjunctOrConjunctQueries(isConjunct bool, queries ...*v1.Query) *v1.Query {
	if len(queries) == 0 {
		return &v1.Query{}
	}

	if len(queries) == 1 {
		return queries[0]
	}
	if isConjunct {
		return &v1.Query{
			Query: &v1.Query_Conjunction{Conjunction: &v1.ConjunctionQuery{Queries: queries}},
		}
	}

	return &v1.Query{
		Query: &v1.Query_Disjunction{Disjunction: &v1.DisjunctionQuery{Queries: queries}},
	}
}

func queryFromBaseQuery(baseQuery *v1.BaseQuery) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{BaseQuery: baseQuery},
	}
}

func stringQuery(value string) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_StringQuery{StringQuery: &v1.StringQuery{Query: value}},
	})
}

func matchFieldQuery(field, value string) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_MatchFieldQuery{MatchFieldQuery: &v1.MatchFieldQuery{Field: field, Value: value}},
	})
}
