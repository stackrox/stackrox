//go:build sql_integration

package m208tom209

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_208_to_m_209_policy_updates_for_4_6/conversion"
	"github.com/stackrox/rox/migrator/migrations/m_208_to_m_209_policy_updates_for_4_6/schema"
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

	// Insert some policies that won't be migrated to set the baseline
	policies := []*storage.Policy{
		simplePolicy(uuid.NewV4().String()),
		simplePolicy(uuid.NewV4().String()),
	}

	for _, p := range policies {
		s.addPolicyToDB(p)
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
			afterPolicy, _ := policymigrationhelper.ReadPolicyFromFile(policyDiffFS, filepath.Join("policies_before_and_after/after", diff.PolicyFileName))
			var foundPolicies []schema.Policies
			result := s.gormDB.Limit(1).Where(&schema.Policies{ID: afterPolicy.GetId()}).Find(&foundPolicies)
			s.Require().NoError(result.Error)
			migratedPolicy, err := conversion.ConvertPolicyToProto(&foundPolicies[0])
			s.Require().NoError(err)
			protoassert.ElementsMatch(s.T(), migratedPolicy.Exclusions, afterPolicy.Exclusions, "exclusion do not match after migration")
			protoassert.ElementsMatch(s.T(), migratedPolicy.PolicySections, afterPolicy.PolicySections, "policy sections do not match after migration")
		})
	}
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
