// Package id provides lightweight ID construction utilities for composite
// primary keys. These are separated from pkg/search/postgres to avoid
// forcing lightweight consumers to import the full postgres search stack
// (which transitively pulls in gorm, inflection, etc).
package id

import "strings"

// Separator is the separator used in IDs constructed from multiple primary keys.
const Separator = "#"

// FromPks creates a composite ID from multiple primary key values.
func FromPks(pks []string) string {
	return strings.Join(pks, Separator)
}

// ToParts splits a composite ID into its primary key parts.
func ToParts(id string) []string {
	return strings.Split(id, Separator)
}
