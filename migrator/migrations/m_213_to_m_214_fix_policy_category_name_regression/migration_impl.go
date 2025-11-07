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

// TODO(dont-merge): generate/write and import any store required for the migration (skip any unnecessary step):
//  - create a schema subdirectory
//  - create a schema/old subdirectory
//  - create a schema/new subdirectory
//  - create a stores subdirectory
//  - create a stores/previous subdirectory
//  - create a stores/updated subdirectory
//  - copy the old schemas from pkg/postgres/schema to schema/old
//  - copy the old stores from their location in central to appropriate subdirectories in stores/previous
//  - generate the new schemas in pkg/postgres/schema and the new stores where they belong
//  - copy the newly generated schemas from pkg/postgres/schema to schema/new
//  - remove the calls to GetSchemaForTable and to RegisterTable from the copied schema files
//  - remove the xxxTableName constant from the copied schema files
//  - copy the newly generated stores from their location in central to appropriate subdirectories in stores/updated
//  - remove any unused function from the copied store files (the minimum for the public API should contain Walk, UpsertMany, DeleteMany)
//  - remove the scoped access control code from the copied store files
//  - remove the metrics collection code from the copied store files

// TODO(dont-merge): Determine if this change breaks a previous releases database.
// If so increment the `MinimumSupportedDBVersionSeqNum` to the `CurrentDBVersionSeqNum` of the release immediately
// following the release that cannot tolerate the change in pkg/migrations/internal/fallback_seq_num.go.
//
// For example, in 4.2 a column `column_v2` is added to replace the `column_v1` column in 4.1.
// All the code from 4.2 onward will not reference `column_v1`. At some point in the future a rollback to 4.1
// will not longer be supported and we want to remove `column_v1`. To do so, we will upgrade the schema to remove
// the column and update the `MinimumSupportedDBVersionSeqNum` to be the value of `CurrentDBVersionSeqNum` in 4.2
// as 4.1 will no longer be supported. The migration process will inform the user of an error when trying to migrate
// to a software version that can no longer be supported by the database.

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

// TODO(dont-merge): Write the additional code to support the migration

// TODO(dont-merge): remove any pending TODO
