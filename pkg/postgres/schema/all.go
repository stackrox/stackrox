package schema

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	log = logging.LoggerForModule()
	// registeredTables is map of sql table name to go schema of the sql table.
	registeredTables = make(map[string]*walker.Schema)
)

// RegisterTable maps a table to an object type for the purposes of metrics gathering
func RegisterTable(schema *walker.Schema) {
	if _, ok := registeredTables[schema.Table]; ok {
		log.Fatalf("table %q is already registered for %s", schema.Table, schema.Type)
		return
	}
	registeredTables[schema.Table] = schema
}

// GetSchemaForTable return the schema registered for specified table name.
func GetSchemaForTable(tableName string) *walker.Schema {
	return registeredTables[tableName]
}

// GetAllRegisteredSchemas returns all registered schemas.
func GetAllRegisteredSchemas() map[string]*walker.Schema {
	ret := make(map[string]*walker.Schema)
	for k, v := range registeredTables {
		ret[k] = v
	}
	return ret
}
