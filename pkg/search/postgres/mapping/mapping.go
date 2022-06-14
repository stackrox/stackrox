package mapping

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/postgres/walker"
)

var (
	categoryToTableMap = make(map[v1.SearchCategory]*walker.Schema)

	log = logging.LoggerForModule()
)

// RegisterCategoryToTable attributes a search category to a table schema.
func RegisterCategoryToTable(category v1.SearchCategory, table *walker.Schema) {
	if val, ok := categoryToTableMap[category]; ok {
		log.Fatalf("Cannot register category %s with table %s, it is already registered with %s", category, table.Table, val.Table)
	}
	categoryToTableMap[category] = table
}

// GetTableFromCategory returns the schema based on the category.
func GetTableFromCategory(category v1.SearchCategory) *walker.Schema {
	return categoryToTableMap[category]
}

// GetAllRegisteredSchemas returns all registered schemas.
func GetAllRegisteredSchemas() []*walker.Schema {
	allSchemas := make([]*walker.Schema, 0, len(categoryToTableMap))
	for _, schema := range categoryToTableMap {
		allSchemas = append(allSchemas, schema)
	}
	return allSchemas
}
