package search

import (
	"fmt"
	"strconv"
	"strings"
)

// QueryBuilder builds a search query
type QueryBuilder struct {
	query map[string][]string
}

// NewQueryBuilder instantiates a query builder with no values
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: make(map[string][]string),
	}
}

// AddString adds a key value pair to the query
func (qb *QueryBuilder) AddString(k, v string) *QueryBuilder {
	qb.query[k] = append(qb.query[k], v)
	return qb
}

// AddBool adds a string key and a bool value pair
func (qb *QueryBuilder) AddBool(k string, v bool) *QueryBuilder {
	qb.query[k] = append(qb.query[k], strconv.FormatBool(v))
	return qb
}

// Query returns the string version of the query
func (qb *QueryBuilder) Query() string {
	pairs := make([]string, 0, len(qb.query))
	for k, values := range qb.query {
		pairs = append(pairs, fmt.Sprintf("%s:%s", k, strings.Join(values, ",")))
	}
	return strings.Join(pairs, "+")
}
