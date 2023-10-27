//go:build sql_integration

package m193tom194

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_193_to_m_194_policy_updates_for_4_3/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
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

func simplePolicy(policyID string) *storage.Policy {
	return &storage.Policy{
		Id:   policyID,
		Name: fmt.Sprintf("Policy with id %s", policyID),
	}
}

func (s *policyMigrationTestSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())

	s.db = pghelper.ForT(s.T(), false)
	s.gormDB = s.db.GetGormDB().WithContext(s.ctx)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePoliciesStmt)

	// insert other un policies that won't be migrated in the db for migration to run successfully
	policies := []*storage.Policy{
		simplePolicy("47cb9e0a-879a-417b-9a8f-de644d7c8a77"),
		simplePolicy("6226d4ad-7619-4a0b-a160-46373cfcee66"),
		simplePolicy("436811e7-892f-4da6-a0f5-8cc459f1b954"),
		simplePolicy("742e0361-bddd-4a2d-8758-f2af6197f61d"),
		simplePolicy("16c95922-08c4-41b6-a721-dc4b2a806632"),
		simplePolicy("a9b9ecf7-9707-4e32-8b62-d03018ed454f"),
		simplePolicy("32d770b9-c6ba-4398-b48a-0c3e807644ed"),
	}

	for _, p := range policies {
		s.addPolicyToDB(p)
	}
}

func (s *policyMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *policyMigrationTestSuite) TestMigration() {

	// Insert the policies to be migrated
	for _, diff := range policyDiffs {
		beforePolicy, err := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/before", diff.PolicyFileName))
		s.Require().NoError(err)
		s.addPolicyToDB(beforePolicy)
	}

	// Run the migration
	s.Require().NoError(migration.Run(&types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.gormDB,
	}))

	// Verify for each
	for _, diff := range policyDiffs {
		s.Run(fmt.Sprintf("Testing policy %s", diff.PolicyFileName), func() {

			afterPolicy, _ := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/after", diff.PolicyFileName))
			var foundPolicies []schema.Policies
			result := s.gormDB.Limit(1).Where(&schema.Policies{ID: afterPolicy.GetId()}).Find(&foundPolicies)
			s.Require().NoError(result.Error)
			migratedPolicy, err := schema.ConvertPolicyToProto(&foundPolicies[0])
			s.Require().NoError(err)
			s.Equal(afterPolicy, migratedPolicy)
		})
	}
}

func (s *policyMigrationTestSuite) addPolicyToDB(policy *storage.Policy) {
	p, err := schema.ConvertPolicyFromProto(policy)
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(p).Error)
}
