package postgres

import "github.com/stackrox/rox/pkg/postgres/walker"

// isChildTableField returns true if the field comes from a child table
// (a table with a foreign key relationship to the main query table)
func isChildTableField(field *walker.Field, schema *walker.Schema) bool {
	if field == nil || schema == nil {
		return false
	}

	// If field's table is the same as the query's main table, it's not a child field
	if field.Schema.Table == schema.Table {
		return false
	}

	// Check if field's table is in the children relationships
	return isChildTable(field.Schema.Table, schema)
}

// isChildTable returns true if the given table name is a child table of the schema
func isChildTable(tableName string, schema *walker.Schema) bool {
	if schema == nil {
		return false
	}

	// Walk through all children of the schema
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
