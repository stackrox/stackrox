package search

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ParseQueryForAutocomplete parses the input string specific for autocomplete requests.
func ParseQueryForAutocomplete(query string) (*v1.Query, string, error) {
	return autocompleteQueryParser{}.parse(query)
}

// ParseQuery parses the input query with the supplied options.
func ParseQuery(query string, opts ...ParseQueryOption) (*v1.Query, error) {
	parser := generalQueryParser{}
	for _, opt := range opts {
		opt(&parser)
	}
	return parser.parse(query)
}

// ParseQueryOption represents an option to use when parsing queries.
type ParseQueryOption func(parser *generalQueryParser)

// LinkFields will parse the input query string as a set of linked fields.
func LinkFields() ParseQueryOption {
	return func(parser *generalQueryParser) {
		parser.LinkFields = true
	}
}

// HighlightFields will cause all fields in the input query to be highlighted in the output query object.
func HighlightFields() ParseQueryOption {
	return func(parser *generalQueryParser) {
		parser.HighlightFields = true
	}
}

// MatchAllIfEmpty will cause an empty query to be returned if the input query is empty (as opposed to an error).
func MatchAllIfEmpty() ParseQueryOption {
	return func(parser *generalQueryParser) {
		parser.MatchAllIfEmpty = true
	}
}

// FilterFields uses a predicate to filter our fields from a raw query based on the field key.
func FilterFields(query string, pred func(field string) bool) string {
	if query == "" {
		return query
	}
	pairs := strings.Split(query, "+")
	pairsToKeep := make([]string, 0, len(pairs))
	for _, pair := range pairs {
		key, _, valid := parsePair(pair, false)
		if !valid {
			continue
		}
		if !pred(key) {
			continue
		}
		pairsToKeep = append(pairsToKeep, pair)
	}
	return strings.Join(pairsToKeep, "+")
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
	if allowEmpty {
		if spl[1] == "" {
			spl[1] = WildcardString
		} else if string(spl[1][len(spl[1])-1]) == "," {
			spl[1] = spl[1] + WildcardString
		}
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
