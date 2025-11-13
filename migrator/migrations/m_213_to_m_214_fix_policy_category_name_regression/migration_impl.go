package m213tom214

import (
	"regexp"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/policy"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/policycategory"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/policycategoryedge"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
)

var (
	uppercaseRegex = regexp.MustCompile("[A-Z]")
)

func migrate(database *types.Databases) error {
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, policycategory.CreateTablePolicyCategoriesStmt)
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, policy.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(database.DBCtx, database.GormDB, policycategoryedge.CreateTablePolicyCategoryEdgesStmt)
	// Use databases.DBCtx to take advantage of the transaction wrapping present in the migration initiator
	db := database.PostgresDB
	categoryStore := policycategory.New(db)
	edgesStore := policycategoryedge.New(db)

	conn, err := db.Acquire(database.DBCtx)
	defer conn.Release()
	if err != nil {
		return err
	}

	categories := make(map[string][]*storage.PolicyCategory, 0)
	err = categoryStore.GetByQueryFn(database.DBCtx, search.EmptyQuery(), func(category *storage.PolicyCategory) error {
		categories[strings.ToLower(category.GetName())] = append(categories[strings.ToLower(category.GetName())], category)
		return nil
	})
	if err != nil {
		return err
	}

	for categoryNameLower, currentCategories := range categories {
		currentCandidate := &storage.PolicyCategory{}
		var categoryIds []string
		for _, category := range currentCategories {
			categoryIds = append(categoryIds, category.GetId())
			if len(uppercaseRegex.FindAllStringIndex(category.GetName(), -1)) >= len(uppercaseRegex.FindAllStringIndex(currentCandidate.GetName(), -1)) {
				currentCandidate = category
			}
		}
		edgesToUpdate := make([]*storage.PolicyCategoryEdge, 0)
		err = edgesStore.GetByQueryFn(database.DBCtx, search.NewQueryBuilder().AddExactMatches(search.PolicyCategoryID, categoryIds...).ProtoQuery(), func(edge *storage.PolicyCategoryEdge) error {
			edge.CategoryId = currentCandidate.GetId()
			edgesToUpdate = append(edgesToUpdate, edge)
			return nil
		})
		if err != nil {
			return err
		}
		err = edgesStore.UpsertMany(database.DBCtx, edgesToUpdate)
		if err != nil {
			return err
		}
		_, err = db.Exec(database.DBCtx, "DELETE FROM policy_categories WHERE LOWER(name) = $1 AND id != $2", categoryNameLower, currentCandidate.GetId())
		if err != nil {
			return err
		}
	}

	return nil
}
