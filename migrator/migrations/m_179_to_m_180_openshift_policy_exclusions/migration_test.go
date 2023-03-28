//go:build sql_integration

package m179tom180

import (
	"context"
	"testing"

	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_179_to_m_180_openshift_policy_exclusions/postgres"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
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

	db          *pghelper.TestPostgres
	policyStore policyPostgresStore.Store
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
	testPolicy.Id = "ed8c7957-14de-40bc-aeab-d27ceeecfa7b"
	testPolicy.Name = "Iptables Executed in Privileged Container"
	testPolicy.Description = "Alert on privileged pods that execute iptables"

	require.NoError(s.T(), s.policyStore.Upsert(ctx, testPolicy))

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	s.NoError(migration.Run(dbs))

	q := search.NewQueryBuilder().AddExactMatches(search.PolicyID, testPolicy.GetId()).ProtoQuery()
	policy, err := s.policyStore.GetByQuery(ctx, q)
	s.NoError(err)
	s.Equal(len(policy[0].Exclusions), 1, "exclusion do not match after migration")

}
