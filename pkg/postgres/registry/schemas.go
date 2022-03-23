package registry

import "github.com/stackrox/rox/pkg/postgres/walker"

// TODO: Move this into globalDB and stitch with individual stores.

var (
	tableNameToSchemaMap = make(map[string]*walker.Schema)
)

// RegisterSchema registers a schema to global record of db schemas.
func RegisterSchema(schema *walker.Schema) {
	panic("not implemented")
}

// GetSchemaForTable return the schema registered for specified table name.
func GetSchemaForTable(tableName string) *walker.Schema {
	return tableNameToSchemaMap[tableName]
}

// GetAllRegisteredSchemas returns all registered schemas.
func GetAllRegisteredSchemas() map[string]*walker.Schema {
	ret := make(map[string]*walker.Schema)
	for k, v := range tableNameToSchemaMap {
		ret[k] = v
	}
	return ret
}
