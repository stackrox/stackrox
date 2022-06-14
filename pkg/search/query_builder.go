package search

import (
	"fmt"
	"sort"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/conv"
	"github.com/stackrox/rox/pkg/generic"
	"github.com/stackrox/rox/pkg/set"
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

	// AtLeastOnePrefix is the prefix to require that all values match the query
	AtLeastOnePrefix = "!!"

	// EqualityPrefixSuffix is the prefix for an exact match
	EqualityPrefixSuffix = `"`
)

var (
	comparatorRepresentation = map[storage.Comparator]string{
		storage.Comparator_LESS_THAN:              "<",
		storage.Comparator_LESS_THAN_OR_EQUALS:    "<=",
		storage.Comparator_EQUALS:                 "",
		storage.Comparator_GREATER_THAN_OR_EQUALS: ">=",
		storage.Comparator_GREATER_THAN:           ">",
	}
)

// ExactMatchString returns the "exact match" form of the query.
func ExactMatchString(query string) string {
	return fmt.Sprintf(`"%s"`, query)
}

// RegexQueryString returns the "regex" form of the query.
func RegexQueryString(query string) string {
	return fmt.Sprintf("%s%s", RegexPrefix, query)
}

// NegateQueryString negates the given query.
func NegateQueryString(query string) string {
	return fmt.Sprintf("%s%s", NegationPrefix, query)
}

// IsNegationQuery returns whether or not this would turn into a negation query
func IsNegationQuery(value string) bool {
	return strings.HasPrefix(value, NegationPrefix)
}

// NumericQueryString converts a numeric query to the string query format.
func NumericQueryString(comparator storage.Comparator, value float32) string {
	return fmt.Sprintf("%s%.2f", comparatorRepresentation[comparator], value)
}

type fieldValue struct {
	l           FieldLabel
	v           string
	highlighted bool
}

// NewPagination create a new pagination object
func NewPagination() *Pagination {
	return &Pagination{
		qp: &v1.QueryPagination{},
	}
}

// Pagination defines the pagination to be used with the query
type Pagination struct {
	qp *v1.QueryPagination
}

// Limit sets the limit
func (p *Pagination) Limit(limit int32) *Pagination {
	p.qp.Limit = limit
	return p
}

// Offset sets the offset
func (p *Pagination) Offset(offset int32) *Pagination {
	p.qp.Offset = offset
	return p
}

// AddSortOption adds the sort option to the pagination object
func (p *Pagination) AddSortOption(option FieldLabel, reversed bool) *Pagination {
	p.qp.SortOptions = append(p.qp.SortOptions, &v1.QuerySortOption{
		Field:    string(option),
		Reversed: reversed,
	})
	return p
}

// QueryBuilder builds a search query
type QueryBuilder struct {
	fieldsToValues map[FieldLabel][]string
	ids            []string
	linkedFields   [][]fieldValue

	highlightedFields map[FieldLabel]struct{}

	pagination *Pagination
}

// NewQueryBuilder instantiates a query builder with no values
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		fieldsToValues:    make(map[FieldLabel][]string),
		highlightedFields: make(map[FieldLabel]struct{}),
	}
}

// WithPagination applies pagination to the query
func (qb *QueryBuilder) WithPagination(p *Pagination) *QueryBuilder {
	qb.pagination = p
	return qb
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

// AddDocIDSet adds the set of ids to the DocID query of the QueryBuilder.
func (qb *QueryBuilder) AddDocIDSet(idSet set.StringSet) *QueryBuilder {
	for id := range idSet {
		qb.ids = append(qb.ids, id)
	}
	return qb
}

// AddLinkedFieldsHighlighted is a convenience wrapper around AddLinkedFields and MarkHighlighted.
func (qb *QueryBuilder) AddLinkedFieldsHighlighted(fields []FieldLabel, values []string) *QueryBuilder {
	return qb.addLinkedFields(fields, values, true)
}

// AddLinkedFieldsWithHighlightValues allows you to add linked fields and specify granuarly which ones you want highlights for.
func (qb *QueryBuilder) AddLinkedFieldsWithHighlightValues(fields []FieldLabel, values []string, highlighted []bool) *QueryBuilder {
	if len(fields) != len(values) || len(fields) != len(highlighted) {
		panic(fmt.Sprintf("Incorrect input to AddLinkedFieldsHighlighted, all three slices (%+v, %+v and %+v) must have the same length", fields, values, highlighted))
	}
	fieldValues := make([]fieldValue, len(fields))
	for i, field := range fields {
		fieldValues[i] = fieldValue{field, values[i], highlighted[i]}
	}
	qb.linkedFields = append(qb.linkedFields, fieldValues)
	return qb
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

// AddDays adds a query on the (timestamp) field k that matches if the value in k
// is at least 'days' days before time.Now.
func (qb *QueryBuilder) AddDays(k FieldLabel, days int64) *QueryBuilder {
	return qb.AddStrings(k, fmt.Sprintf(">%dd", days))
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

// AddExactMatches adds a key value pair to the query
func (qb *QueryBuilder) AddExactMatches(k FieldLabel, values ...string) *QueryBuilder {
	for _, v := range values {
		qb.fieldsToValues[k] = append(qb.fieldsToValues[k], ExactMatchString(v))
	}
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
	bools := conv.FormatBool(v...)

	qb.fieldsToValues[k] = append(qb.fieldsToValues[k], bools...)
	return qb
}

// AddNumericField adds a numeric field.
func (qb *QueryBuilder) AddNumericField(k FieldLabel, comparator storage.Comparator, value float32) *QueryBuilder {
	return qb.AddStrings(k, NumericQueryString(comparator, value))
}

// AddNumericFieldHighlighted is a convenience wrapper to AddNumericField and MarkHighlighted.
func (qb *QueryBuilder) AddNumericFieldHighlighted(k FieldLabel, comparator storage.Comparator, value float32) *QueryBuilder {
	return qb.AddNumericField(k, comparator, value).MarkHighlighted(k)
}

// AddGenericTypeLinkedFields allows you to add linked fields of different types.
func (qb *QueryBuilder) AddGenericTypeLinkedFields(fields []FieldLabel, values []interface{}) *QueryBuilder {
	strValues := make([]string, 0, len(values))
	for _, value := range values {
		strValues = append(strValues, generic.String(value))
	}
	return qb.addLinkedFields(fields, strValues, false)
}

// AddGenericTypeLinkedFieldsHighligted allows you to add linked fields of different types and MarkHighlighted.
func (qb *QueryBuilder) AddGenericTypeLinkedFieldsHighligted(fields []FieldLabel, values []interface{}) *QueryBuilder {
	strValues := make([]string, 0, len(values))
	for _, value := range values {
		strValues = append(strValues, generic.String(value))
	}
	return qb.addLinkedFields(fields, strValues, true)
}

// Query returns the string version of the query.
func (qb *QueryBuilder) Query() string {
	pairs := make([]string, 0, len(qb.fieldsToValues))
	for k, values := range qb.fieldsToValues {
		pairs = append(pairs, fmt.Sprintf("%s:%s", k, strings.Join(values, ",")))
	}
	sort.Strings(pairs)
	return strings.Join(pairs, "+")
}

// ProtoQuery generates a proto query from the query
func (qb *QueryBuilder) ProtoQuery() *v1.Query {
	queries := make([]*v1.Query, 0, len(qb.fieldsToValues)+len(qb.linkedFields))

	if len(qb.ids) > 0 {
		queries = append(queries, docIDQuery(qb.ids))
	}

	// Sort the queries by field value, to ensure consistency of output.
	fields := qb.getSortedFields()

	for _, field := range fields {
		_, highlighted := qb.highlightedFields[field]
		queries = append(queries, queryFromFieldValues(field.String(), qb.fieldsToValues[field], highlighted))
	}

	for _, linkedFieldsGroup := range qb.linkedFields {
		queries = append(queries, matchLinkedFieldsQuery(linkedFieldsGroup))
	}

	cq := ConjunctionQuery(queries...)
	if qb.pagination != nil {
		cq.Pagination = qb.pagination.qp
	}
	return cq
}

func (qb *QueryBuilder) getSortedFields() []FieldLabel {
	fields := make([]FieldLabel, 0, len(qb.fieldsToValues))
	for field := range qb.fieldsToValues {
		fields = append(fields, field)
	}
	return SortFieldLabels(fields)
}

// RawQuery returns raw query in string form
func (qb *QueryBuilder) RawQuery() (string, error) {
	var query string
	for field, values := range qb.fieldsToValues {
		if query != "" {
			query += "+"
		}
		q := strings.Join(values, ",")
		query += fmt.Sprintf("%s:%s", field, q)
	}
	return query, nil
}

// EmptyQuery is a shortcut function to receive an empty query, to avoid requiring having to create an empty query builder.
func EmptyQuery() *v1.Query {
	return &v1.Query{}
}

// MatchNoneQuery returns a v1.Query that maps to a bleve query that does not match any results
func MatchNoneQuery() *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BaseQuery{
			BaseQuery: &v1.BaseQuery{
				Query: &v1.BaseQuery_MatchNoneQuery{},
			},
		},
	}
}

// NewBooleanQuery takes in a must conjunction query and a must not disjunction query
func NewBooleanQuery(must *v1.ConjunctionQuery, mustNot *v1.DisjunctionQuery) *v1.Query {
	return &v1.Query{
		Query: &v1.Query_BooleanQuery{
			BooleanQuery: &v1.BooleanQuery{
				Must:    must,
				MustNot: mustNot,
			},
		},
	}
}
