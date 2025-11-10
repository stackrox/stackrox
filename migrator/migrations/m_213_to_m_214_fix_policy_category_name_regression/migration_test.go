//go:build sql_integration

package m213tom214

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenPolicySchema "github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/policy"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/policycategory"
	"github.com/stackrox/rox/migrator/migrations/m_213_to_m_214_fix_policy_category_name_regression/policycategoryedge"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type migrationTestSuite struct {
	suite.Suite

	db     *pghelper.TestPostgres
	ctx    context.Context
	gormDB *gorm.DB
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	s.gormDB = s.db.GetGormDB().WithContext(s.ctx)

	// Create the schemas and tables required for the pre-migration dataset
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), frozenPolicySchema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), policycategory.CreateTablePolicyCategoriesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), policycategoryedge.CreateTablePolicyCategoryEdgesStmt)
}

func (s *migrationTestSuite) TestMigration() {
	// Create 5 policies
	policyIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		policyID := uuid.NewV4().String()
		policyIDs[i] = policyID
		policy := &storage.Policy{
			Id:   policyID,
			Name: "Test Policy " + uuid.NewV4().String(),
		}
		policySchema, err := frozenPolicySchema.ConvertPolicyFromProto(policy)
		s.Require().NoError(err)
		s.Require().NoError(s.gormDB.Create(policySchema).Error)
	}

	// Create 5 policy categories with varying degrees of capitalization
	// All have the same name when lowercased: "security best practices"
	// "SECURITY BEST PRACTICES" has the most uppercase letters (21)
	categoryNames := []string{
		"Security Best Practices", // 3 uppercase: S, B, P
		"security best practices", // 0 uppercase
		"SECURITY BEST PRACTICES", // 21 uppercase (should win)
		"Security best practices", // 1 uppercase: S
		"SeCuRiTy BeSt PrAcTiCeS", // 11 uppercase
	}
	categoryIDs := make([]string, 5)
	mostCapitalizedCategoryID := ""
	mostCapitalizedCategoryName := "SECURITY BEST PRACTICES"

	for i, name := range categoryNames {
		categoryID := uuid.NewV4().String()
		categoryIDs[i] = categoryID
		if name == mostCapitalizedCategoryName {
			mostCapitalizedCategoryID = categoryID
		}
		category := &storage.PolicyCategory{
			Id:   categoryID,
			Name: name,
		}
		categorySchema, err := policycategory.ConvertPolicyCategoryFromProto(category)
		s.Require().NoError(err)
		s.Require().NoError(s.gormDB.Create(categorySchema).Error)
	}

	// Create 5 policy category edges, one for each policy pointing to a different category
	for i, policyID := range policyIDs {
		edgeID := uuid.NewV4().String()
		edge := &storage.PolicyCategoryEdge{
			Id:         edgeID,
			PolicyId:   policyID,
			CategoryId: categoryIDs[i],
		}
		edgeSchema, err := policycategoryedge.ConvertPolicyCategoryEdgeFromProto(edge)
		s.Require().NoError(err)
		s.Require().NoError(s.gormDB.Create(edgeSchema).Error)
	}

	// Run the migration
	dbs := &types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}
	s.Require().NoError(migration.Run(dbs))

	// Verify that all edges now point to the most capitalized category
	var edges []*policycategoryedge.PolicyCategoryEdges
	result := s.gormDB.Find(&edges)
	s.Require().NoError(result.Error)
	s.Require().Len(edges, 5, "Should have 5 edges after migration")

	for _, edge := range edges {
		s.Equal(mostCapitalizedCategoryID, edge.CategoryID,
			"All edges should point to the most capitalized category")
	}

	// Verify that the inferior categories were deleted
	var categories []*policycategory.PolicyCategories
	result = s.gormDB.Find(&categories)
	s.Require().NoError(result.Error)
	s.Require().Len(categories, 1, "Should have only 1 category after migration (the most capitalized one)")
	s.Equal(mostCapitalizedCategoryID, categories[0].ID, "The remaining category should be the most capitalized one")
	s.Equal(mostCapitalizedCategoryName, categories[0].Name, "The remaining category should have the most capitalized name")
}
