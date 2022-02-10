package mapping

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	categoryToTableMap = make(map[v1.SearchCategory]string)
	tableToCategoryMap = make(map[string]v1.SearchCategory)

	log = logging.LoggerForModule()
)

func RegisterCategoryToTable(category v1.SearchCategory, table string) {
	if val, ok := categoryToTableMap[category]; ok {
		log.Fatalf("Cannot register category %s with table %s, it is already registered with %s", category, table, val)
	}
	categoryToTableMap[category] = table
	tableToCategoryMap[table] = category
}

func GetTableFromCategory(category v1.SearchCategory) string {
	return categoryToTableMap[category]
}

func GetCategoryFromTable(table string) v1.SearchCategory {
	return tableToCategoryMap[table]
}
