//go:build sql_integration

package m170tom171

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_172_to_m_173_openshift_policy_exclusion/postgres"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type categoriesMigrationTestSuite struct {
	suite.Suite

	db            *pghelper.TestPostgres
	policyStore   policyPostgresStore.Store
	categoryStore policyCategoryPostgresStore.Store
	edgeStore     policyCategoryEdgePostgresStore.Store
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(categoriesMigrationTestSuite))
}

func (s *categoriesMigrationTestSuite) SetupTest() {
	s.db = pghelper.ForT(s.T(), true)
	s.policyStore = policyPostgresStore.New(s.db.DB)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), frozenSchema.CreateTablePoliciesStmt)

}

func (s *categoriesMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *categoriesMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())
	testPolicy := fixtures.GetPolicy()
	exclusion := &storage.Exclusion{
		Name:       "Existing exclusion 1",
		Deployment: &storage.Exclusion_Deployment{Scope: &storage.Scope{Namespace: "test-namespace-1"}}})
	policy.Exclusions = append(policy.Exclusions, exclusion)

	require.NoError(s.T(), s.policyStore.Upsert(ctx, testPolicy))

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	s.NoError(migration.Run(dbs))

	q := search.NewQueryBuilder().AddExactMatches(search.PolicyID, testPolicy.GetId()).ProtoQuery()
	policy, err := s.policyStore.GetByQuery(ctx, q)
	s.NoError(err)
	s.Equal(policy.Exclusions, exclusion, "exclusion do not match after migration")

}
