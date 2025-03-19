//go:build sql_integration

package m198tom199

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_198_to_m_199_policy_description_and_criteria_updates/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

type policyMigrationTestSuite struct {
	suite.Suite

	db *pghelper.TestPostgres

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

func (s *policyMigrationTestSuite) addPolicyToDB(policy *storage.Policy) {
	p, err := schema.ConvertPolicyFromProto(policy)
	s.Require().NoError(err)
	s.Require().NoError(s.db.GetGormDB().Create(p).Error)
}

func (s *policyMigrationTestSuite) SetupTest() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePoliciesStmt)

	// insert other un policies that won't be migrated in the db for migration to run successfully
	policies := []*storage.Policy{
		simplePolicy("47cb9e0a-879a-417b-9a8f-de644d7c8a77"),
		simplePolicy("657f4d37-55ab-42f2-bdce-9a4b74a67328"),
		simplePolicy("cf80fb33-c7d0-4490-b6f4-e56e1f27b4e4"),
		simplePolicy("742e0361-bddd-4a2d-8758-f2af6197f61d"),
		simplePolicy("16c95922-08c4-41b6-a721-dc4b2a806632"),
		simplePolicy("32d770b9-c6ba-4398-b48a-0c3e807644ed"),
	}

	for _, p := range policies {
		s.addPolicyToDB(p)
	}
}

func (s *policyMigrationTestSuite) TestMigration() {

	// Insert the policies to be migrated
	for _, diff := range policyDiffs {
		beforePolicy, err := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/before", diff.PolicyFileName))
		s.Require().NoError(err)
		s.addPolicyToDB(beforePolicy)
	}

	s.Require().NoError(migration.Run(&types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}))

	// Verify for each
	gormDB := s.db.GetGormDB()
	for _, diff := range policyDiffs {
		s.Run(fmt.Sprintf("Testing policy %s", diff.PolicyFileName), func() {

			afterPolicy, _ := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/after", diff.PolicyFileName))
			afterPolicy.Categories = nil
			var foundPolicies []schema.Policies
			result := gormDB.Limit(1).Where(&schema.Policies{ID: afterPolicy.GetId()}).Find(&foundPolicies)
			s.Require().NoError(result.Error)
			migratedPolicy, err := schema.ConvertPolicyToProto(&foundPolicies[0])
			s.Require().NoError(err)
			protoassert.Equal(s.T(), afterPolicy, migratedPolicy)
		})
	}

}
