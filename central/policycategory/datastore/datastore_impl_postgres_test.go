//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	policyCategorySearch "github.com/stackrox/rox/central/policycategory/search"
	pgStore "github.com/stackrox/rox/central/policycategory/store/postgres"
	edgeDataStore "github.com/stackrox/rox/central/policycategoryedge/datastore"
	policyCategoryEdgeSearch "github.com/stackrox/rox/central/policycategoryedge/search"
	policyCategoryEdgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestPolicyCategoryDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyCategoryPostgresDataStoreTestSuite))
}

type PolicyCategoryPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx           context.Context
	db            postgres.DB
	gormDB        *gorm.DB
	datastore     DataStore
	edgeDatastore edgeDataStore.DataStore
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) SetupSuite() {

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := postgres.New(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) SetupTest() {
	pgStore.Destroy(s.ctx, s.db)
	policyCategoryEdgePostgres.Destroy(s.ctx, s.db)

	policyCategoryEdgeStorage := policyCategoryEdgePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	policyCategoryEdgeIndexer := policyCategoryEdgePostgres.NewIndexer(s.db)
	policyCategorySearcher := policyCategoryEdgeSearch.New(policyCategoryEdgeStorage, policyCategoryEdgeIndexer)
	s.edgeDatastore = edgeDataStore.New(policyCategoryEdgeStorage, policyCategorySearcher)

	policyCategoryStore := pgStore.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	policyCategoryIndexer := pgStore.NewIndexer(s.db)
	s.datastore = New(policyCategoryStore, policyCategorySearch.New(policyCategoryStore, policyCategoryIndexer), s.edgeDatastore)
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *PolicyCategoryPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	category := &storage.PolicyCategory{
		Id:        "id-1",
		Name:      "Boo's Category",
		IsDefault: false,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration),
		))

	// Add category.
	_, err := s.datastore.AddPolicyCategory(ctx, category)
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, category.GetName()).ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	// Add new category.
	anotherCategory := &storage.PolicyCategory{
		Id:        "id-2",
		Name:      "Boo's Other Category",
		IsDefault: false,
	}
	_, err = s.datastore.AddPolicyCategory(ctx, anotherCategory)
	s.NoError(err)

	// Search multiple images.
	categories, err := s.datastore.SearchRawPolicyCategories(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(categories, 2)

	// Search for just one category.
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, category.GetName()).ProtoQuery()
	categories, err = s.datastore.SearchRawPolicyCategories(ctx, q)
	s.NoError(err)
	s.Len(categories, 1)
	s.Equal("id-1", categories[0].GetId())

}
