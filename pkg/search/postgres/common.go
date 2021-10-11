package postgres

import (
	"database/sql"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	categoryToTableMap = make(map[v1.SearchCategory]string)

	log = logging.LoggerForModule()
)

func RegisterCategoryToTable(category v1.SearchCategory, table string) {
	if val, ok := categoryToTableMap[category]; ok {
		log.Fatalf("Cannot register category %s with table %s, it is already registered with %s", category, table, val)
	}
	categoryToTableMap[category] = table
}

func RunSearchRequest(category v1.SearchCategory, q *v1.Query, db *sql.DB, optionsMap searchPkg.OptionsMap) ([]searchPkg.Result, error) {
	return nil, nil
}

func RunCountRequest(category v1.SearchCategory, q *v1.Query, db *sql.DB, optionsMap searchPkg.OptionsMap) (int, error) {
	return 0, nil
}
