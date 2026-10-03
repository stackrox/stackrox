//go:build sql_integration

package m227tom228

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_227_to_m_228_harden_secret_env_var_policy/conversion"
	"github.com/stackrox/rox/migrator/migrations/m_227_to_m_228_harden_secret_env_var_policy/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type migrationTestSuite struct {
	suite.Suite

	db     *pghelper.TestPostgres
	gormDB *gorm.DB

	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	s.gormDB = s.db.GetGormDB().WithContext(s.ctx)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePolicyCategoriesStmt)

	// Insert some policies that won't be migrated to set the baseline
	policies := []*storage.Policy{
		simplePolicy(uuid.NewV4().String()),
		simplePolicy(uuid.NewV4().String()),
	}

	for _, p := range policies {
		s.addPolicyToDB(p)
	}
}

// SetupTest removes the migrated policy before each test so tests are
// independent of execution order (each test inserts its own copy).
func (s *migrationTestSuite) SetupTest() {
	for _, diff := range policyDiffs {
		beforePolicy, err := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/before", diff.PolicyFileName))
		s.Require().NoError(err)
		s.Require().NoError(s.gormDB.Where(&schema.Policies{ID: beforePolicy.GetId()}).Delete(&schema.Policies{}).Error)
	}
}

func (s *migrationTestSuite) TestMigration() {
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

	// Verify for each policy
	for _, diff := range policyDiffs {
		s.Run(fmt.Sprintf("Testing policy %s", diff.PolicyFileName), func() {
			afterPolicy, err := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/after", diff.PolicyFileName))
			s.Require().NoError(err)
			migratedPolicy := s.getPolicyFromDB(afterPolicy.GetId())
			protoassert.ElementsMatch(s.T(), migratedPolicy.GetPolicySections(), afterPolicy.GetPolicySections(), "policy sections do not match after migration")
			s.Equal(afterPolicy.GetDescription(), migratedPolicy.GetDescription(), "description does not match after migration")
		})
	}
}

// TestMigrationSkipsUserModifiedPolicy verifies that if a user has changed a
// compared field (here: the policy name), the migration leaves the policy
// untouched and does not apply the hardened criteria or description.
func (s *migrationTestSuite) TestMigrationSkipsUserModifiedPolicy() {
	for _, diff := range policyDiffs {
		beforePolicy, err := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/before", diff.PolicyFileName))
		s.Require().NoError(err)
		beforePolicy.Name += " (user modified)"
		s.addPolicyToDB(beforePolicy)
	}

	s.Require().NoError(migration.Run(&types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.gormDB,
	}))

	for _, diff := range policyDiffs {
		s.Run(fmt.Sprintf("Testing policy %s", diff.PolicyFileName), func() {
			beforePolicy, err := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/before", diff.PolicyFileName))
			s.Require().NoError(err)
			migratedPolicy := s.getPolicyFromDB(beforePolicy.GetId())

			// Migration must be skipped: sections and description stay as in "before".
			protoassert.ElementsMatch(s.T(), beforePolicy.GetPolicySections(), migratedPolicy.GetPolicySections(), "modified policy should keep its original policy sections")
			s.Equal(beforePolicy.GetDescription(), migratedPolicy.GetDescription(), "modified policy should keep its original description")
		})
	}
}

func (s *migrationTestSuite) getPolicyFromDB(id string) *storage.Policy {
	var foundPolicies []schema.Policies
	result := s.gormDB.Limit(1).Where(&schema.Policies{ID: id}).Find(&foundPolicies)
	s.Require().NoError(result.Error)
	s.Require().Len(foundPolicies, 1)
	migratedPolicy, err := conversion.ConvertPolicyToProto(&foundPolicies[0])
	s.Require().NoError(err)
	return migratedPolicy
}

func simplePolicy(policyID string) *storage.Policy {
	return &storage.Policy{
		Id:   policyID,
		Name: fmt.Sprintf("Policy with id %s", policyID),
	}
}

func (s *migrationTestSuite) addPolicyToDB(policy *storage.Policy) {
	p, err := conversion.ConvertPolicyFromProto(policy)
	s.Require().NoError(err)
	s.Require().NoError(s.gormDB.Create(p).Error)
}
