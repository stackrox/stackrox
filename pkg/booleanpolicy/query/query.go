package query

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/mapeval"
)

// An Operator denotes how to combine multiple values.
//go:generate stringer -type=Operator
type Operator int

// This block enumerates valid operators.
const (
	Unset Operator = iota
	And
	Or
)

// FieldQuery is a base query, consisting of a query on a specific field.
// This corresponds to a PolicyGroup.
type FieldQuery struct {
	Field    string
	Values   []string
	Operator Operator
	Negate   bool
	MatchAll bool
}

// A Query represents a query.
// This corresponds to a single policy section.
type Query struct {
	FieldQueries []*FieldQuery
}

// SimpleMatchFieldQuery is a convenience function that constructs a simple query
// that matches just the field and value given.
func SimpleMatchFieldQuery(field, value string) *Query {
	return &Query{FieldQueries: []*FieldQuery{
		{Field: field, Values: []string{value}},
	}}
}

// ShouldMatchIfAllOf builds a conjunction of all query groups.
func ShouldMatchIfAllOf(queryGroups ...string) string {
	return strings.Join(queryGroups, mapeval.ConjunctionMarker)
}

// ShouldContain builds a query that matches a map if it contains the particular key value pair. Key/Value could be
// left empty (using "").
func ShouldContain(key, value string) string {
	return string(fmt.Sprintf("%s=%s", key, value))
}

// ShouldNotContain builds a query that matches a map if it does not contain a particular key value pair.
// Key/Value could be left empty (using "").
func ShouldNotContain(key, value string) string {
	return fmt.Sprintf("%s%s=%s", mapeval.ShouldNotMatchMarker, key, value)
}

// CompileMapQuery builds a disjunction of all query groups.
func CompileMapQuery(queryGroups ...string) string {
	return strings.Join(queryGroups, mapeval.DisjunctionMarker)
}
