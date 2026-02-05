package postgres

import "github.com/stackrox/rox/pkg/postgres/walker"

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
