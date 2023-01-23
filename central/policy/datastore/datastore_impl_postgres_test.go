//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	notifierDSMocks "github.com/stackrox/rox/central/notifier/datastore/mocks"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store/postgres"
	policyCategoryDS "github.com/stackrox/rox/central/policycategory/datastore"
	categorySearch "github.com/stackrox/rox/central/policycategory/search"
	categoryPostgres "github.com/stackrox/rox/central/policycategory/store/postgres"
	policyCategoryEdgeDS "github.com/stackrox/rox/central/policycategoryedge/datastore"
	edgeSearch "github.com/stackrox/rox/central/policycategoryedge/search"
	edgePostgres "github.com/stackrox/rox/central/policycategoryedge/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestPolicyDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(PolicyPostgresDataStoreTestSuite))
}

type PolicyPostgresDataStoreTestSuite struct {
	suite.Suite

	ctx            context.Context
	db             *pgxpool.Pool
	gormDB         *gorm.DB
	mockClusterDS  *clusterDSMocks.MockDataStore
	mockNotifierDS *notifierDSMocks.MockDataStore

	datastore  DataStore
	categoryDS policyCategoryDS.DataStore
	edgeDS     policyCategoryEdgeDS.DataStore
}

func (s *PolicyPostgresDataStoreTestSuite) SetupSuite() {
	s.T().Setenv(features.NewPolicyCategories.EnvVar(), "true")
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() || !features.NewPolicyCategories.Enabled() {
		s.T().Skip("Skipping. This test requires postgres and categories flag enabled.")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool
}

func (s *PolicyPostgresDataStoreTestSuite) SetupTest() {
	postgres.Destroy(s.ctx, s.db)
	categoryPostgres.Destroy(s.ctx, s.db)
	edgePostgres.Destroy(s.ctx, s.db)

	s.mockClusterDS = clusterDSMocks.NewMockDataStore(gomock.NewController(s.T()))
	s.mockNotifierDS = notifierDSMocks.NewMockDataStore(gomock.NewController(s.T()))

	categoryStorage := categoryPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	categoryIndex := categoryPostgres.NewIndexer(s.db)
	categorySearcher := categorySearch.New(categoryStorage, categoryIndex)

	edgeStorage := edgePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	edgeIndex := edgePostgres.NewIndexer(s.db)
	edgeSearcher := edgeSearch.New(edgeStorage, edgeIndex)

	s.categoryDS = policyCategoryDS.New(categoryStorage, categoryIndex, categorySearcher, policyCategoryEdgeDS.New(edgeStorage, edgeIndex, edgeSearcher))

	policyStore := postgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	policyIndex := postgres.NewIndexer(s.db)
	s.datastore = New(policyStore, policyIndex, search.New(policyStore, policyIndex), s.mockClusterDS, s.mockNotifierDS, s.categoryDS)

}

func (s *PolicyPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
}

func (s *PolicyPostgresDataStoreTestSuite) TestInsertUpdatePolicy() {
	policy := fixtures.GetPolicy()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Policy, resources.Cluster),
	))

	// Add policy.
	_, err := s.datastore.AddPolicy(ctx, policy)
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	policy.Categories = []string{"Image Assurance", "Boo Category 1", "Boo Category 2"}
	// Update policy
	s.NoError(s.datastore.UpdatePolicy(ctx, policy))

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Container Configuration").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)

	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Boo Category 1").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	// Delete policy
	s.NoError(s.datastore.RemovePolicy(ctx, policy.GetId()))
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Boo Category 1").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *PolicyPostgresDataStoreTestSuite) TestImportPolicy() {

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Policy, resources.Cluster),
	))
	s.mockClusterDS.EXPECT().GetClusters(ctx).Return([]*storage.Cluster{fixtures.GetCluster("cluster-1")}, nil)

	policy := fixtures.GetPolicy()
	policy.Id = ""

	// Import policy.
	_, allSucceeded, err := s.datastore.ImportPolicies(ctx, []*storage.Policy{policy}, true)
	s.NoError(err)
	s.True(allSucceeded)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)

	// Delete policy
	s.NoError(s.datastore.RemovePolicy(ctx, policy.GetId()))
	q = pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *PolicyPostgresDataStoreTestSuite) TestSearchPolicyCategoryFeatureDisabled() {
	s.T().Setenv(features.NewPolicyCategories.EnvVar(), "false")

	// Policy should get upserted with category names stored inside the policy storage proto object
	// no edges, no separate category objects)
	policy := fixtures.GetPolicy()

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Policy, resources.Cluster),
	))

	// Add policy.
	_, err := s.datastore.AddPolicy(ctx, policy)
	s.NoError(err)

	// Basic unscoped search.
	results, err := s.datastore.Search(ctx, pkgSearch.EmptyQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Category, "Image Assurance").ProtoQuery()
	results, err = s.datastore.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
	s.Equal(policy.GetId(), results[0].ID)
}
