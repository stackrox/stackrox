package pgutils

import (
	"crypto/fips140"

	"gorm.io/gorm/schema"
)

// fipsNamingStrategy wraps schema.NamingStrategy to allow GORM's
// non-cryptographic SHA-1 identifier hashing under GODEBUG=fips140=only.
type fipsNamingStrategy struct {
	schema.NamingStrategy
}

func (ns fipsNamingStrategy) RelationshipFKName(rel schema.Relationship) string {
	var name string
	fips140.WithoutEnforcement(func() {
		name = ns.NamingStrategy.RelationshipFKName(rel)
	})
	return name
}

func (ns fipsNamingStrategy) CheckerName(table, column string) string {
	var name string
	fips140.WithoutEnforcement(func() {
		name = ns.NamingStrategy.CheckerName(table, column)
	})
	return name
}

func (ns fipsNamingStrategy) IndexName(table, column string) string {
	var name string
	fips140.WithoutEnforcement(func() {
		name = ns.NamingStrategy.IndexName(table, column)
	})
	return name
}

func (ns fipsNamingStrategy) UniqueName(table, column string) string {
	var name string
	fips140.WithoutEnforcement(func() {
		name = ns.NamingStrategy.UniqueName(table, column)
	})
	return name
}
