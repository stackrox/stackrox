//go:build sql_integration

package m226tom227

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_226_to_m_227_exclude_rhacm_sa_kubeadmin_pol/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

var (
	existingSAs = []string{
		"system:serviceaccount:openshift-authentication-operator:authentication-operator",
		"system:apiserver",
		"system:serviceaccount:openshift-authentication:oauth-openshift",
		"system:serviceaccount:openshift-compliance:api-resource-collector",
		"system:serviceaccount:openshift-oauth-apiserver:oauth-apiserver-sa",
	}
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
}

func (s *migrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *migrationTestSuite) SetupTest() {
	_, err := s.db.Exec(s.ctx, "DELETE FROM policies")
	s.Require().NoError(err)
}

func (s *migrationTestSuite) TestMigrationAddsRHACMSA() {
	policy := buildPolicy(existingSAs)
	s.insertPolicy(policy)

	s.runMigration()

	updated := s.getPolicy()
	group := findUserNameGroup(updated)
	s.Require().NotNil(group)
	s.Require().Len(group.GetValues(), 6)
	s.Equal(rhacmSA, group.GetValues()[5].GetValue())
}

func (s *migrationTestSuite) TestMigrationIsIdempotent() {
	sasWithRHACM := append(existingSAs, rhacmSA)
	policy := buildPolicy(sasWithRHACM)
	s.insertPolicy(policy)

	s.runMigration()

	updated := s.getPolicy()
	group := findUserNameGroup(updated)
	s.Require().NotNil(group)
	s.Require().Len(group.GetValues(), 6, "should not duplicate the RHACM SA")
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
		"SELECT serialized FROM policies WHERE id = $1", policyID).Scan(&serialized)
	s.Require().NoError(err)

	policy := &storage.Policy{}
	s.Require().NoError(policy.UnmarshalVT(serialized))
	return policy
}

func buildPolicy(excludedSAs []string) *storage.Policy {
	values := make([]*storage.PolicyValue, len(excludedSAs))
	for i, sa := range excludedSAs {
		values[i] = &storage.PolicyValue{Value: sa}
	}

	return &storage.Policy{
		Id:          policyID,
		Name:        "OpenShift: Kubeadmin Secret Accessed",
		Description: "Alert when the kubeadmin secret is accessed",
		Severity:    storage.Severity_HIGH_SEVERITY,
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Kubernetes Resource",
						Values:    []*storage.PolicyValue{{Value: "SECRETS"}},
					},
					{
						FieldName: "Kubernetes API Verb",
						Values:    []*storage.PolicyValue{{Value: "GET"}},
					},
					{
						FieldName: "Kubernetes Resource Name",
						Values:    []*storage.PolicyValue{{Value: "kubeadmin"}},
					},
					{
						FieldName: "Kubernetes User Name",
						Negate:    true,
						Values:    values,
					},
				},
			},
		},
	}
}
