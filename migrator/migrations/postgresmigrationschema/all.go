package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

/**
 * This package keeps a snapshot of the Postgres schema
 * when we migrate to use Central DB (Postgres). The Database
 * schema is only used to migrate data from legacy databases.
 */

// RegisterTable mocks the schema registration.
func RegisterTable(_ *walker.Schema, _ *postgres.CreateStmts) {
}

// GetSchemaForTable mocks getting schema for a table
func GetSchemaForTable(_ string) *walker.Schema {
	return &walker.Schema{}
}
