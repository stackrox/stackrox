package search

import (
	"errors"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ParseRawQueryOrEmpty is a convenience wrapper around ParseRawQuery which returns the empty
// proto query instead of erroring out if an empty string is passed.
func ParseRawQueryOrEmpty(query string) (*v1.Query, error) {
	if query == "" {
		return EmptyQuery(), nil
	}
	return parseRawQuery(query, false)
}

// ParseRawQuery takes the text based query and converts to the query proto.
// It expects the received query to be non-empty.
func ParseRawQuery(query string) (*v1.Query, error) {
	if query == "" {
		return nil, errors.New("empty query received")
	}
	return parseRawQuery(query, false)
}

// ParseRawQueryIntoMap returns a map[search key] []values from a string query
func ParseRawQueryIntoMap(query string) map[string][]string {
	pairs := strings.Split(query, "+")
	resultMap := make(map[string][]string)

	for _, pair := range pairs {
		key, commaSeparatedValues, valid := parsePair(pair, false)
		if !valid {
			continue
		}
		resultMap[key] = strings.Split(commaSeparatedValues, ",")
	}
	return resultMap
}

// ParseAutocompleteRawQuery parses the query, but extends the last value with .* and highlights
func ParseAutocompleteRawQuery(query string) (*v1.Query, error) {
	return parseRawQuery(query, true)
}

func parseRawQuery(query string, isAutocompleteQuery bool) (*v1.Query, error) {
	pairs := strings.Split(query, "+")

	queries := make([]*v1.Query, 0, len(pairs))
	for i, pair := range pairs {
		key, commaSeparatedValues, valid := parsePair(pair, isAutocompleteQuery)
		if !valid {
			continue
		}
		if i == len(pairs)-1 && isAutocompleteQuery {
			queries = append(queries, queryFromFieldValues(key, strings.Split(commaSeparatedValues, ","), true))

		} else {
			queries = append(queries, queryFromKeyValue(key, commaSeparatedValues))
		}
	}

	// We always want to return an error here, because it means that the query is ill-defined.
	if len(queries) == 0 {
		return nil, errors.New("after parsing, query is empty")
	}

	return ConjunctionQuery(queries...), nil
}

func queryFromKeyValue(key, commaSeparatedValues string) *v1.Query {
	valueSlice := strings.Split(commaSeparatedValues, ",")

	return queryFromFieldValues(key, valueSlice, false)
}

// Extracts "key", "value1,value2" from a string in the format key:value1,value2
func parsePair(pair string, allowEmpty bool) (key string, values string, valid bool) {
	pair = strings.TrimSpace(pair)
	if len(pair) == 0 {
		return
	}

	spl := strings.SplitN(pair, ":", 2)
	// len < 2 implies there isn't a colon and the second check verifies that the : wasn't the last char
	if len(spl) < 2 || (spl[1] == "" && !allowEmpty) {
		return
	}
	// If empty strings are allowed, it means we're treating them as wildcards.
	if spl[1] == "" {
		spl[1] = WildcardString
	}
	return spl[0], spl[1], true
}

func queryFromFieldValues(field string, values []string, highlight bool) *v1.Query {
	queries := make([]*v1.Query, 0, len(values))
	for _, value := range values {
		queries = append(queries, matchFieldQuery(field, value, highlight))
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

func matchFieldQuery(field, value string, highlight bool) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_MatchFieldQuery{MatchFieldQuery: &v1.MatchFieldQuery{Field: field, Value: value, Highlight: highlight}},
	})
}

func matchLinkedFieldsQuery(fieldValues []fieldValue) *v1.Query {
	mfqs := make([]*v1.MatchFieldQuery, len(fieldValues))
	for i, fv := range fieldValues {
		mfqs[i] = &v1.MatchFieldQuery{Field: fv.l.String(), Value: fv.v, Highlight: fv.highlighted}
	}

	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_MatchLinkedFieldsQuery{MatchLinkedFieldsQuery: &v1.MatchLinkedFieldsQuery{
			Query: mfqs,
		}},
	})
}

func docIDQuery(ids []string) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_DocIdQuery{DocIdQuery: &v1.DocIDQuery{Ids: ids}},
	})
}
