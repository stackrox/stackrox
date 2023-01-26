//go:build sql_integration

package m170tom171

import (
	"context"
	"testing"

	policyCategoryEdgePostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategoryedgepostgresstore"
	policyCategoryPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policycategorypostgresstore"
	policyPostgresStore "github.com/stackrox/rox/migrator/migrations/m_170_to_m_171_create_policy_categories_and_edges/policypostgresstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/schema"
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
	s.policyStore = policyPostgresStore.New(s.db.Pool)
	s.categoryStore = policyCategoryPostgresStore.New(s.db.Pool)
	s.edgeStore = policyCategoryEdgePostgresStore.New(s.db.Pool)
	schema.ApplySchemaForTable(context.Background(), s.db.GetGormDB(), schema.PoliciesTableName)
	schema.ApplySchemaForTable(context.Background(), s.db.GetGormDB(), schema.PolicyCategoriesTableName)

}

func (s *categoriesMigrationTestSuite) TearDownTest() {
	s.db.Teardown(s.T())
}

func (s *categoriesMigrationTestSuite) TestMigration() {
	ctx := sac.WithAllAccess(context.Background())
	testPolicy := fixtures.GetPolicy()
	testPolicy.Categories = []string{"Test Category"}

	require.NoError(s.T(), s.policyStore.Upsert(ctx, testPolicy))

	dbs := &types.Databases{
		PostgresDB: s.db.Pool,
		GormDB:     s.db.GetGormDB(),
	}

	s.NoError(migration.Run(dbs))

	q := search.NewQueryBuilder().AddExactMatches(search.PolicyCategoryName, testPolicy.GetCategories()[0]).ProtoQuery()
	categoriesAfterMigration, err := s.categoryStore.GetByQuery(ctx, q)
	s.NoError(err)
	s.Len(categoriesAfterMigration, 1)
	s.Equal(categoriesAfterMigration[0].GetName(), "Test Category", "categories do not match after migration")

	edges, err := s.edgeStore.GetAll(ctx)
	s.NoError(err)
	s.Len(edges, 1)
}
