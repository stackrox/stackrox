package m213tom214

import (
	"regexp"
	"strings"

	"github.com/cloudflare/cfssl/log"
	"github.com/lib/pq"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/schema"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
)

func migrate(database *types.Databases) error {
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTablePolicyCategoriesStmt)
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, schema.CreateTablePolicyCategoryEdgesStmt)
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	db := database.PostgresDB

	conn, err := db.Acquire(database.DBCtx)
	if err != nil {
		return err
	}

	rows, err := conn.Query(database.DBCtx, "SELECT id,name FROM policy_categories WHERE LOWER(name) in (SELECT LOWER(name) FROM policy_categories GROUP BY LOWER(name) HAVING COUNT(name) > 1);")
	if err != nil {
		return err
	}

	categories, err := readRows(rows)
	if err != nil {
		return err
	}

	uppercaseRegex := regexp.MustCompile("[A-Z]")
	for categoryNameLower, currentCategories := range categories {
		currentCandidate := &schema.PolicyCategories{}
		var categoryIds []string
		for _, category := range currentCategories {
			categoryIds = append(categoryIds, category.ID)
			if len(uppercaseRegex.FindAllStringIndex(category.Name, -1)) >= len(uppercaseRegex.FindAllStringIndex(currentCandidate.Name, -1)) {
				currentCandidate = category
			}
		}
		_, err = db.Exec(database.DBCtx, "UPDATE policy_category_edges SET categoryid = $1 WHERE categoryid = ANY($2::text[])", currentCandidate.ID, pq.Array(categoryIds))
		if err != nil {
			return err
		}
		_, err = db.Exec(database.DBCtx, "DELETE FROM policy_categories WHERE LOWER(name) = $1 AND id != $2", categoryNameLower, currentCandidate.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func readRows(rows *postgres.Rows) (map[string][]*schema.PolicyCategories, error) {
	res := make(map[string][]*schema.PolicyCategories)

	for rows.Next() {
		var id string
		var name string

		if err := rows.Scan(&id, &name); err != nil {
			log.Errorf("Error scanning row: %v", err)
		}

		category := &schema.PolicyCategories{
			ID:   id,
			Name: name,
		}
		res[strings.ToLower(name)] = append(res[strings.ToLower(name)], category)
	}

	return res, rows.Err()
}
