//go:build sql_integration

package m206tom207

import (
	"context"
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_206_to_m_207_bug_fix_default_policy_category/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type policyMigrationTestSuite struct {
	suite.Suite

	db     *pghelper.TestPostgres
	gormDB *gorm.DB

	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(policyMigrationTestSuite))
}

func (s *policyMigrationTestSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())

	s.db = pghelper.ForT(s.T(), false)
	s.gormDB = s.db.GetGormDB().WithContext(s.ctx)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePolicyCategoriesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePolicyCategoryEdgesStmt)
}

func (s *policyMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *policyMigrationTestSuite) TestMigration() {

	// Run the migration
	s.Require().NoError(migration.Run(&types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.gormDB,
	}))

	// Verify for each
	for _, diff := range policyDiffs {
		s.Run(fmt.Sprintf("Testing policy %s", diff.PolicyFileName), func() {
			var foundEdge []schema.PolicyCategoryEdges
			result := s.gormDB.Limit(1).Where(&schema.PolicyCategoryEdges{ID: afterPolicy.GetId()}).Find(&foundPolicies)
			s.Require().NoError(result.Error)
			migratedPolicy, err := schema.ConvertPolicyToProto(&foundPolicies[0])
			s.Require().NoError(err)
			s.ElementsMatch(afterPolicy.Categories, migratedPolicy.Categories)
		})
	}
}

func (s *policyMigrationTestSuite) addPolicyToDB(policy *storage.Policy) {
	p, err := schema.ConvertPolicyFromProto(policy)
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(p).Error)
}
