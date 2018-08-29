package search

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// QueryBuilder builds a search query
type QueryBuilder struct {
	query map[string][]string
	raw   string
}

// NewQueryBuilder instantiates a query builder with no values
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: make(map[string][]string),
	}
}

// AddStrings adds a key value pair to the query
func (qb *QueryBuilder) AddStrings(k string, v ...string) *QueryBuilder {
	qb.query[k] = append(qb.query[k], v...)
	return qb
}

// AddBools adds a string key and a bool value pair
func (qb *QueryBuilder) AddBools(k string, v ...bool) *QueryBuilder {
	bools := make([]string, 0, len(v))
	for _, b := range v {
		bools = append(bools, strconv.FormatBool(b))
	}
	qb.query[k] = append(qb.query[k], bools...)
	return qb
}

// AddStringQuery adds a raw string query
func (qb *QueryBuilder) AddStringQuery(v string) *QueryBuilder {
	qb.raw = v
	return qb
}

// Query returns the string version of the query
func (qb *QueryBuilder) Query() string {
	pairs := make([]string, 0, len(qb.query))
	for k, values := range qb.query {
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
	queries := make([]*v1.Query, 0, len(qb.query))

	// Sort the queries by field value, to ensure consistency of output.
	fields := make([]string, 0, len(qb.query))
	for field := range qb.query {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	for _, field := range fields {
		queries = append(queries, queryFromFieldValues(field, qb.query[field]))
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
