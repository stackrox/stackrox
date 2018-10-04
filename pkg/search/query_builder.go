package search

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

const (
	// RegexPrefix is the prefix for regex queries.
	RegexPrefix = "r/"

	// WildcardString represents the string we use for wildcard queries.
	WildcardString = "*"

	// NullString represents the string we use for querying for the absence of any value in a field.
	NullString = "-"

	// NegationPrefix is the prefix to negate a query.
	NegationPrefix = "!"
)

var (
	comparatorRepresentation = map[v1.Comparator]string{
		v1.Comparator_LESS_THAN:              "<",
		v1.Comparator_LESS_THAN_OR_EQUALS:    "<=",
		v1.Comparator_EQUALS:                 "",
		v1.Comparator_GREATER_THAN_OR_EQUALS: ">=",
		v1.Comparator_GREATER_THAN:           ">",
	}
)

// RegexQueryString returns the "regex" form of the query.
func RegexQueryString(query string) string {
	return fmt.Sprintf("%s%s", RegexPrefix, strings.ToLower(query))
}

// NegateQueryString negates the given query.
func NegateQueryString(query string) string {
	return fmt.Sprintf("%s%s", NegationPrefix, query)
}

// NullQueryString returns a null query
func NullQueryString() string {
	return NullString
}

// NumericQueryString converts a numeric query to the string query format.
func NumericQueryString(comparator v1.Comparator, value float32) string {
	return fmt.Sprintf("%s%.2f", comparatorRepresentation[comparator], value)
}

type fieldValue struct {
	l           FieldLabel
	v           string
	highlighted bool
}

// QueryBuilder builds a search query
type QueryBuilder struct {
	fieldsToValues map[FieldLabel][]string
	ids            []string
	linkedFields   [][]fieldValue

	raw               string
	highlightedFields map[FieldLabel]struct{}
}

// NewQueryBuilder instantiates a query builder with no values
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		fieldsToValues:    make(map[FieldLabel][]string),
		highlightedFields: make(map[FieldLabel]struct{}),
	}
}

// AddLinkedRegexesHighlighted adds linked regexes.
func (qb *QueryBuilder) AddLinkedRegexesHighlighted(fields []FieldLabel, values []string) *QueryBuilder {
	regexValues := make([]string, len(values))
	for i, value := range values {
		regexValues[i] = RegexQueryString(value)
	}
	return qb.AddLinkedFieldsHighlighted(fields, regexValues)
}

// AddLinkedFields adds a bunch of fields and values where the matches must be in corresponding places in both fields.
// For example, if you have an []struct{a string, b string}, and you query for "a": "avalue" and "b": "bvalue",
// then the following slice would normally match.
// []{{"a": "avalue", "b": "NOTbvalue"}, {"a": "NOTavalue", "b": "bvalue"}
// But this function specifies that the query must be on linked fields,
// so that an array would match ONLY if it had {"a": "avalue", "b": "bvalue"} on the same element.
func (qb *QueryBuilder) AddLinkedFields(fields []FieldLabel, values []string) *QueryBuilder {
	return qb.addLinkedFields(fields, values, false)
}

// AddDocIDs adds the list of ids to the DocID query of the QueryBuilder.
func (qb *QueryBuilder) AddDocIDs(ids ...string) *QueryBuilder {
	qb.ids = append(qb.ids, ids...)
	return qb
}

// AddLinkedFieldsHighlighted is a convenience wrapper around AddLinkedFields and MarkHighlighted.
func (qb *QueryBuilder) AddLinkedFieldsHighlighted(fields []FieldLabel, values []string) *QueryBuilder {
	return qb.addLinkedFields(fields, values, true)
}

func (qb *QueryBuilder) addLinkedFields(fields []FieldLabel, values []string, highlighted bool) *QueryBuilder {
	if len(fields) != len(values) {
		panic("Incorrect input to AddLinkedFields, the two slices must have the same length")
	}
	fieldValues := make([]fieldValue, len(fields))
	for i, field := range fields {
		fieldValues[i] = fieldValue{field, values[i], highlighted}
	}

	qb.linkedFields = append(qb.linkedFields, fieldValues)
	return qb
}

// AddDaysHighlighted is a convenience wrapper around AddDays and MarkHighlighted.
func (qb *QueryBuilder) AddDaysHighlighted(k FieldLabel, days int64) *QueryBuilder {
	return qb.AddDays(k, days).MarkHighlighted(k)
}

// AddDays adds a query on the (timestamp) field k that matches the value in k
// is at least 'days' days before time.Now.
func (qb *QueryBuilder) AddDays(k FieldLabel, days int64) *QueryBuilder {
	return qb.AddStrings(k, fmt.Sprintf("<=%dd", days))
}

// MarkHighlighted marks the field as one that we want results to be highlighted for.
func (qb *QueryBuilder) MarkHighlighted(k FieldLabel) *QueryBuilder {
	qb.highlightedFields[k] = struct{}{}
	return qb
}

// AddStringsHighlighted is a convenience wrapper to add a key value pair and mark
// the field as highlighted.
func (qb *QueryBuilder) AddStringsHighlighted(k FieldLabel, v ...string) *QueryBuilder {
	return qb.AddStrings(k, v...).MarkHighlighted(k)
}

// AddNullField adds a very for documents that don't contain the specified field.
func (qb *QueryBuilder) AddNullField(k FieldLabel) *QueryBuilder {
	return qb.AddStrings(k, NullString)
}

// AddStrings adds a key value pair to the query.
func (qb *QueryBuilder) AddStrings(k FieldLabel, v ...string) *QueryBuilder {
	qb.fieldsToValues[k] = append(qb.fieldsToValues[k], v...)
	return qb
}

// AddMapQuery adds a query for a key and a value in a map field.
func (qb *QueryBuilder) AddMapQuery(k FieldLabel, mapKey, mapValue string) *QueryBuilder {
	qb.AddStrings(k, fmt.Sprintf("%s=%s", mapKey, mapValue))
	return qb
}

// AddRegexesHighlighted is a convenience wrapper to add regexes and mark the field as highlighted.
func (qb *QueryBuilder) AddRegexesHighlighted(k FieldLabel, regexes ...string) *QueryBuilder {
	return qb.AddRegexes(k, regexes...).MarkHighlighted(k)
}

// AddRegexes adds regexes to match on the field.
func (qb *QueryBuilder) AddRegexes(k FieldLabel, regexes ...string) *QueryBuilder {
	for _, r := range regexes {
		qb.fieldsToValues[k] = append(qb.fieldsToValues[k], RegexQueryString(r))
	}
	return qb
}

// AddBoolsHighlighted is a convenience wrapper to AddBools and MarkHighlighted.
func (qb *QueryBuilder) AddBoolsHighlighted(k FieldLabel, bools ...bool) *QueryBuilder {
	return qb.AddBools(k, bools...).MarkHighlighted(k)
}

// AddBools adds a string key and a bool value pair.
func (qb *QueryBuilder) AddBools(k FieldLabel, v ...bool) *QueryBuilder {
	bools := make([]string, 0, len(v))
	for _, b := range v {
		bools = append(bools, strconv.FormatBool(b))
	}

	qb.fieldsToValues[k] = append(qb.fieldsToValues[k], bools...)
	return qb
}

// AddStringQuery adds a raw string query.
func (qb *QueryBuilder) AddStringQuery(v string) *QueryBuilder {
	qb.raw = v
	return qb
}

// AddNumericField adds a numeric field.
func (qb *QueryBuilder) AddNumericField(k FieldLabel, comparator v1.Comparator, value float32) *QueryBuilder {
	return qb.AddStrings(k, NumericQueryString(comparator, value))
}

// AddNumericFieldHighlighted is a convenience wrapper to AddNumericField and MarkHighlighted.
func (qb *QueryBuilder) AddNumericFieldHighlighted(k FieldLabel, comparator v1.Comparator, value float32) *QueryBuilder {
	return qb.AddNumericField(k, comparator, value).MarkHighlighted(k)
}

// Query returns the string version of the query.
func (qb *QueryBuilder) Query() string {
	pairs := make([]string, 0, len(qb.fieldsToValues))
	for k, values := range qb.fieldsToValues {
		pairs = append(pairs, fmt.Sprintf("%s:%s", k, strings.Join(values, ",")))
	}
	sort.Strings(pairs)
	if qb.raw != "" {
		return fmt.Sprintf("Has:%s+", qb.raw) + strings.Join(pairs, "+")
	}
	return strings.Join(pairs, "+")
}

// ProtoQuery generates a proto query from the query
func (qb *QueryBuilder) ProtoQuery() *v1.Query {
	queries := make([]*v1.Query, 0, len(qb.fieldsToValues)+len(qb.linkedFields))

	if len(qb.ids) > 0 {
		queries = append(queries, docIDQuery(qb.ids))
	}

	// Sort the queries by field value, to ensure consistency of output.
	fields := make([]FieldLabel, 0, len(qb.fieldsToValues))
	for field := range qb.fieldsToValues {
		fields = append(fields, field)
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i] < fields[j]
	})

	for _, field := range fields {
		_, highlighted := qb.highlightedFields[field]
		queries = append(queries, queryFromFieldValues(field.String(), qb.fieldsToValues[field], highlighted))
	}

	for _, linkedFieldsGroup := range qb.linkedFields {
		queries = append(queries, matchLinkedFieldsQuery(linkedFieldsGroup))
	}

	if qb.raw != "" {
		queries = append(queries, stringQuery(qb.raw))
	}
	return ConjunctionQuery(queries...)
}

// EmptyQuery is a shortcut function to receive an empty query, to avoid requiring having to create an empty query builder.
func EmptyQuery() *v1.Query {
	return &v1.Query{}
}
