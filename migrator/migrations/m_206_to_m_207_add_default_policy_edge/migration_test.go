//go:build sql_integration

package m206tom207

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_206_to_m_207_add_default_policy_edge/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

var (
	policy_category_map = map[string][]string{
		"fb8f8732-c31d-496b-8fb1-d5abe6056e27": {"f732f1a5-1515-4e9e-9179-3ab2aefe9ad9", "99cfb323-c9d3-4e0c-af64-4d0101659866"},
		"ed8c7957-14de-40bc-aeab-d27ceeecfa7b": {"99cfb323-c9d3-4e0c-af64-4d0101659866", "9d924f5d-6679-4449-8154-795449c8e754"},
		"6226d4ad-7619-4a0b-a160-46373cfcee66": {"d2bbe19e-3009-4a0e-a701-a0b621b319a0"},
		"dce17697-1b72-49d2-b18a-05d893cd9368": {"d2bbe19e-3009-4a0e-a701-a0b621b319a0"},
		"a9b9ecf7-9707-4e32-8b62-d03018ed454f": {"99cfb323-c9d3-4e0c-af64-4d0101659866"},
	}
)

type migrationTestSuite struct {
	suite.Suite
	gormDB *gorm.DB

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	s.gormDB = s.db.GetGormDB().WithContext(s.ctx)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePolicyCategoryEdgesStmt)
	categoryAdded := []string{}
	for policyID, categories := range policy_category_map {
		policy := &storage.Policy{
			Id:   policyID,
			Name: fmt.Sprintf("test-policy-%s", uuid.NewV4().String()),
		}

		s.addPolicyToDB(policy)
		for _, categoryID := range categories {
			if slices.Contains(categoryAdded, categoryID) {
				continue
			}
			category := &storage.PolicyCategory{
				Id:   categoryID,
				Name: fmt.Sprintf("test-category-%s", uuid.NewV4().String()),
			}
			s.addCategoryToDB(category)
			categoryAdded = append(categoryAdded, categoryID)
		}
	}
}

func (s *migrationTestSuite) TestMigration() {
	// Run the migration
	s.Require().NoError(migration.Run(&types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.gormDB,
	}))

	// Verify for each edge
	for policyID := range policy_category_map {
		s.Run(fmt.Sprintf("Testing policy %s", policyID), func() {
			var foundEdge []*schema.PolicyCategoryEdges
			result := s.gormDB.Where(&schema.PolicyCategoryEdges{PolicyID: policyID}).Find(&foundEdge)
			s.Require().NoError(result.Error)
			actualCategories := []string{}
			for _, edge := range foundEdge {
				migratedEdge, err := schema.ConvertPolicyCategoryEdgeToProto(edge)
				s.Require().NoError(err)
				actualCategories = append(actualCategories, migratedEdge.CategoryId)
			}
			s.Require().ElementsMatch(actualCategories, policy_category_map[policyID])
		})
	}
}

func (s *migrationTestSuite) addPolicyToDB(policy *storage.Policy) {
	p, err := schema.ConvertPolicyFromProto(policy)
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(p).Error)
}

func (s *migrationTestSuite) addCategoryToDB(category *storage.PolicyCategory) {
	c, err := schema.ConvertPolicyCategoryFromProto(category)
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(c).Error)
}
