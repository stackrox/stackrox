//go:build sql_integration

package m226tom227

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_226_to_m_227_exclude_rhacm_sa_kubeadmin_pol/schema"
	"github.com/stackrox/rox/migrator/migrations/policymigrationhelper"
	categorySchema "github.com/stackrox/rox/migrator/migrations/policymigrationhelper/categorypostgresstorefortest/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

const (
	testPolicyID = "18cbcb62-7d18-4a6c-b2ca-dd1242746943"
	rhacmSA      = "system:serviceaccount:open-cluster-management-agent-addon:config-policy-controller-sa"
)

type migrationTestSuite struct {
	suite.Suite

	db  *pghelper.TestPostgres
	ctx context.Context
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(migrationTestSuite))
}

func (s *migrationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), schema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(s.ctx, s.db.GetGormDB(), categorySchema.CreateTablePolicyCategoriesStmt)
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) SetupTest() {
	_, err := s.db.Exec(s.ctx, "DELETE FROM policies")
	s.Require().NoError(err)
}

func (s *migrationTestSuite) TestMigrationAddsSAToUnmodifiedPolicy() {
	beforePolicy := s.readBeforePolicy()
	s.insertPolicy(beforePolicy)

	s.runMigration()

	updated := s.getPolicy()
	group := findNegatedUserNameGroup(updated)
	s.Require().NotNil(group, "negated Kubernetes User Name group should exist")
	s.Assert().True(hasValue(group, rhacmSA), "RHACM SA should be added")
	s.Assert().Len(group.GetValues(), 6)
}

func (s *migrationTestSuite) TestMigrationIsIdempotent() {
	beforePolicy := s.readBeforePolicy()
	s.insertPolicy(beforePolicy)

	s.runMigration()
	s.runMigration()

	updated := s.getPolicy()
	group := findNegatedUserNameGroup(updated)
	s.Require().NotNil(group)

	count := 0
	for _, v := range group.GetValues() {
		if v.GetValue() == rhacmSA {
			count++
		}
	}
	s.Assert().Equal(1, count, "RHACM SA should appear exactly once")
}

func (s *migrationTestSuite) TestMigrationSkipsUserModifiedPolicy() {
	beforePolicy := s.readBeforePolicy()
	beforePolicy.GetPolicySections()[0].GetPolicyGroups()[0].GetValues()[0].Value = "CONFIGMAPS"
	s.insertPolicy(beforePolicy)

	s.runMigration()

	updated := s.getPolicy()
	group := findNegatedUserNameGroup(updated)
	s.Require().NotNil(group)
	s.Assert().False(hasValue(group, rhacmSA), "should not modify user-customized policy")
	s.Assert().Equal("CONFIGMAPS", updated.GetPolicySections()[0].GetPolicyGroups()[0].GetValues()[0].GetValue(),
		"user modification should be preserved")
}

func (s *migrationTestSuite) TestMigrationSkipsMissingPolicy() {
	s.Require().NoError(migration.Run(&types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}))
}

func (s *migrationTestSuite) runMigration() {
	s.T().Helper()
	s.Require().NoError(migration.Run(&types.Databases{
		GormDB:     s.db.GetGormDB(),
		PostgresDB: s.db.DB,
		DBCtx:      s.ctx,
	}))
}

func (s *migrationTestSuite) readBeforePolicy() *storage.Policy {
	s.T().Helper()
	policy, err := policymigrationhelper.ReadPolicyFromFile(
		policyDiffFS, "policies_before_and_after/before/access_kubeadmin_secret.json")
	s.Require().NoError(err)
	return policy
}

func (s *migrationTestSuite) insertPolicy(policy *storage.Policy) {
	s.T().Helper()
	serialized, err := policy.MarshalVT()
	s.Require().NoError(err)

	_, err = s.db.Exec(s.ctx,
		"INSERT INTO policies (id, name, serialized) VALUES ($1, $2, $3)",
		policy.GetId(), policy.GetName(), serialized)
	s.Require().NoError(err)
}

func (s *migrationTestSuite) getPolicy() *storage.Policy {
	s.T().Helper()
	var serialized []byte
	err := s.db.QueryRow(s.ctx,
		"SELECT serialized FROM policies WHERE id = $1", testPolicyID).Scan(&serialized)
	s.Require().NoError(err)

	policy := &storage.Policy{}
	s.Require().NoError(policy.UnmarshalVT(serialized))
	return policy
}

func findNegatedUserNameGroup(policy *storage.Policy) *storage.PolicyGroup {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if group.GetFieldName() == "Kubernetes User Name" && group.GetNegate() {
				return group
			}
		}
	}
	return nil
}

func hasValue(group *storage.PolicyGroup, value string) bool {
	for _, v := range group.GetValues() {
		if v.GetValue() == value {
			return true
		}
	}
	return false
}
