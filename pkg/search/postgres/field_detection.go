package postgres

import (
	"strings"

	"github.com/stackrox/rox/pkg/postgres/walker"
)

// isChildTableField returns true if the field comes from a child table
func isChildTableField(field *walker.Field, schema *walker.Schema) bool {
	if field == nil || schema == nil {
		return false
	}

	if field.Schema.Table == schema.Table {
		return false
	}

	return isChildTable(field.Schema.Table, schema)
}

// isChildTable returns true if the given table name is a child table of the schema
func isChildTable(tableName string, schema *walker.Schema) bool {
	if schema == nil {
		return false
	}

	for _, child := range schema.Children {
		if child.Table == tableName {
			return true
		}
		// Recursively check nested children (grandchildren, etc.)
		if isChildTable(tableName, child) {
			return true
		}
	}

	return false
}

// FieldNameToDBAlias converts a search field name to the expected db tag alias format.
// This mapping is used to match search field names to struct db tags for automatic
// child table aggregation. The conversion lowercases the field name and joins words
// with underscores.
//
// IMPORTANT: This naming convention couples the search field names to struct db tags.
// If either the schema field names or struct db tags change, this mapping may break.
// Changes to field naming should be accompanied by corresponding test updates to
// validate the mapping remains correct.
//
// Example: "Secret Names" -> "secret_names"
func FieldNameToDBAlias(fieldName string) string {
	return strings.Join(strings.Fields(strings.ToLower(fieldName)), "_")
}
