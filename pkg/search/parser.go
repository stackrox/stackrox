package search

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
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

// MatchAllIfEmpty will cause an empty query to be returned if the input query is empty (as opposed to an error).
func MatchAllIfEmpty() ParseQueryOption {
	return func(parser *generalQueryParser) {
		parser.MatchAllIfEmpty = true
	}
}

// ExcludeFieldLabel removes a specific options key from the search if it exists
func ExcludeFieldLabel(k FieldLabel) ParseQueryOption {
	return func(parser *generalQueryParser) {
		if parser.ExcludedFieldLabels == nil {
			parser.ExcludedFieldLabels = set.NewStringSet()
		}
		parser.ExcludedFieldLabels.Add(k.String())
	}
}

// FilterFields uses a predicate to filter our fields from a raw query based on the field key.
func FilterFields(query string, pred func(field string) bool) string {
	if query == "" {
		return query
	}
	pairs := splitQuery(query)
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

// GetFieldValueFromQuery returns the value associated with given field in the raw query. The bool is true if given field is found in query, else false.
func GetFieldValueFromQuery(query string, label FieldLabel) (string, bool) {
	if query == "" {
		return "", false
	}
	pairs := splitQuery(query)
	for _, pair := range pairs {
		key, val, valid := parsePair(pair, false)
		if !valid {
			continue
		}
		if key == label.String() {
			return val, true
		}
	}
	return "", false
}

func splitQuery(query string) []string {
	var pairs []string
	var previousEnd, previousPlusIndex int
	insideDoubleQuotes := false
	for i, rune := range query {
		if rune == '"' {
			insideDoubleQuotes = !insideDoubleQuotes
			continue
		}
		if insideDoubleQuotes {
			continue
		}
		if rune == ':' && previousPlusIndex != 0 {
			if previousEnd > previousPlusIndex {
				continue
			}

			pairs = append(pairs, query[previousEnd:previousPlusIndex])
			previousEnd = previousPlusIndex + 1
			continue
		}
		if rune == '+' {
			previousPlusIndex = i
		}
	}
	pairs = append(pairs, query[previousEnd:])
	return pairs
}

func splitCommaSeparatedValues(commaSeparatedValues string) []string {
	var vals []string
	start := 0
	insideDoubleQuotes := false
	for i, char := range commaSeparatedValues {
		if char == '"' {
			insideDoubleQuotes = !insideDoubleQuotes
			continue
		}
		if char == ',' && !insideDoubleQuotes {
			vals = append(vals, commaSeparatedValues[start:i])
			start = i + 1
		}
	}
	if start <= len(commaSeparatedValues)-1 {
		vals = append(vals, commaSeparatedValues[start:])
	} else if commaSeparatedValues[len(commaSeparatedValues)-1] == ',' {
		vals = append(vals, "")
	}
	return vals
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
	return strings.TrimSpace(spl[0]), strings.TrimSpace(spl[1]), true
}

func queryFromFieldValues(field string, values []string, highlight bool) *v1.Query {
	// A SQL query can have no more than 65535 parameters.
	if len(values) > MaxQueryParameters {
		log.Errorf("UNEXPECTED: too many parameters %d for a query.  No more than %d parameters allowed in single query", len(values), MaxQueryParameters)
	}
	queries := make([]*v1.Query, 0, len(values))
	for _, value := range values {
		queries = append(queries, MatchFieldQuery(field, value, highlight))
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

// FilterQueryByQuery returns a new Query object where the baseQuery is filtered by the filterQuery.
func FilterQueryByQuery(baseQuery, filterQuery *v1.Query) *v1.Query {
	filteredQuery := ConjunctionQuery(baseQuery, filterQuery)
	filteredQuery.Pagination = baseQuery.GetPagination()

	return filteredQuery
}

// MatchFieldQuery returns a match field query.
// It's a simple convenience wrapper around initializing the struct.
func MatchFieldQuery(field, value string, highlight bool) *v1.Query {
	return queryFromBaseQuery(&v1.BaseQuery{
		Query: &v1.BaseQuery_MatchFieldQuery{MatchFieldQuery: &v1.MatchFieldQuery{Field: field, Value: value, Highlight: highlight}},
	})
}

// matchLinkedFieldsQuery returns a query that matches
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

// QueryModifier describes the query modifiers for a specific individual query
//
//go:generate stringer -type=QueryModifier
type QueryModifier int

// These are the currently supported modifiers
const (
	AtLeastOne QueryModifier = iota
	Negation
	Regex
	Equality
)

// GetValueAndModifiersFromString parses the raw value string into its value and modifiers
func GetValueAndModifiersFromString(value string) (string, []QueryModifier) {
	var queryModifiers []QueryModifier
	trimmedValue := value
	// We only allow at most one modifier from the set {atleastone, negation}.
	// Anything more, we treat as part of the string to query for.
	var negationOrAtLeastOneFound bool
forloop:
	for {
		switch {
		// AtLeastOnePrefix is !! so it must come before negation prefix
		case !negationOrAtLeastOneFound && strings.HasPrefix(trimmedValue, AtLeastOnePrefix) && len(trimmedValue) > len(AtLeastOnePrefix):
			trimmedValue = trimmedValue[len(AtLeastOnePrefix):]
			queryModifiers = append(queryModifiers, AtLeastOne)
			negationOrAtLeastOneFound = true
		case !negationOrAtLeastOneFound && strings.HasPrefix(trimmedValue, NegationPrefix) && len(trimmedValue) > len(NegationPrefix):
			trimmedValue = trimmedValue[len(NegationPrefix):]
			queryModifiers = append(queryModifiers, Negation)
			negationOrAtLeastOneFound = true
		case strings.HasPrefix(trimmedValue, RegexPrefix) && len(trimmedValue) > len(RegexPrefix):
			trimmedValue = strings.ToLower(trimmedValue[len(RegexPrefix):])
			queryModifiers = append(queryModifiers, Regex)
			break forloop // Once we see that it's a regex, we don't check for special-characters in the rest of the string.
		case strings.HasPrefix(trimmedValue, EqualityPrefixSuffix) && strings.HasSuffix(trimmedValue, EqualityPrefixSuffix) && len(trimmedValue) >= 2*len(EqualityPrefixSuffix):
			trimmedValue = trimmedValue[len(EqualityPrefixSuffix) : len(trimmedValue)-len(EqualityPrefixSuffix)]
			queryModifiers = append(queryModifiers, Equality)
			break forloop // Once it's within quotes, we take the value inside as is, and don't try to extract modifiers.
		default:
			break forloop
		}
	}
	return trimmedValue, queryModifiers
}
