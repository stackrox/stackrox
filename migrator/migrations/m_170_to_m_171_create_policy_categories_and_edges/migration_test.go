//go:build sql_integration

package m170tom171

import (
	"context"
	"testing"

	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	policyCategoryEdgePostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategoryedgepostgresstore"
	policyCategoryPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategorypostgresstore"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policypostgresstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
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
	s.db = pghelper.ForT(s.T(), false)
	s.policyStore = policyPostgresStore.New(s.db.DB)
	s.categoryStore = policyCategoryPostgresStore.New(s.db.DB)
	s.edgeStore = policyCategoryEdgePostgresStore.New(s.db.DB)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), frozenSchema.CreateTablePoliciesStmt)
	pgutils.CreateTableFromModel(context.Background(), s.db.GetGormDB(), frozenSchema.CreateTablePolicyCategoriesStmt)

}

func (s *categoriesMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *categoriesMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())
	testPolicy := fixtures.GetPolicy()
	testPolicy.Categories = []string{"Test Category", "test category", "", "Anomalous Activity"}
	require.NoError(s.T(), s.policyStore.Upsert(ctx, testPolicy))

	testPolicy2 := fixtures.GetPolicy()
	testPolicy2.Id = uuid.NewV4().String()
	testPolicy2.Name = "testpolicy2"
	testPolicy2.Categories = []string{"Test Category", "test category", "", "net new category", "Net new category"}
	s.Require().NoError(s.policyStore.Upsert(ctx, testPolicy2))

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
		GormDB:     s.db.GetGormDB(),
	}

	s.NoError(migration.Run(dbs))

	q := search.NewQueryBuilder().AddExactMatches(search.PolicyCategoryName, testPolicy.Categories[0]).ProtoQuery()
	categoriesAfterMigration, err := s.categoryStore.GetByQuery(ctx, q)
	s.NoError(err)
	s.Len(categoriesAfterMigration, 1)
	s.Equal(categoriesAfterMigration[0].GetName(), "Test Category", "categories do not match after migration")

	edges, err := s.edgeStore.GetAll(ctx)
	s.NoError(err)
	s.Len(edges, 4)
}
