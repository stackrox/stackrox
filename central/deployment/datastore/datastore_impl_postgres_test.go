//go:build sql_integration
// +build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jackc/pgx/v4/pgxpool"
	deploymentPostgres "github.com/stackrox/rox/central/deployment/store/postgres"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestDeploymentDataStoreWithPostgres(t *testing.T) {
	suite.Run(t, new(DeploymentPostgresDataStoreTestSuite))
}

type DeploymentPostgresDataStoreTestSuite struct {
	suite.Suite

	mockCtrl            *gomock.Controller
	ctx                 context.Context
	db                  *pgxpool.Pool
	gormDB              *gorm.DB
	imageDatastore      imageDataStore.DataStore
	deploymentDatastore DataStore
	riskDataStore       *riskMocks.MockDataStore
	envIsolator         *envisolator.EnvIsolator
}

func (s *DeploymentPostgresDataStoreTestSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.Require().NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.db = pool

	imagePostgres.Destroy(s.ctx, s.db)
	deploymentPostgres.Destroy(s.ctx, s.db)

	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	ds := imageDataStore.NewWithPostgres(imagePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB, false), imagePostgres.NewIndexer(s.db), s.riskDataStore, ranking.ImageRanker(), ranking.ComponentRanker())
	s.imageDatastore = ds

	s.mockCtrl = gomock.NewController(s.T())
	s.riskDataStore = riskMocks.NewMockDataStore(s.mockCtrl)

	s.deploymentDatastore = newDataStore(
		deploymentPostgres.NewFullTestStore(s.T(), deploymentPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)),
		nil, s.db, nil, nil, nil, s.imageDatastore, nil, nil, s.riskDataStore,
		nil, nil, ranking.ClusterRanker(), ranking.NamespaceRanker(), ranking.DeploymentRanker())
}

func (s *DeploymentPostgresDataStoreTestSuite) TearDownSuite() {
	s.db.Close()
	pgtest.CloseGormDB(s.T(), s.gormDB)
	s.mockCtrl.Finish()
	s.envIsolator.RestoreAll()
}

func (s *DeploymentPostgresDataStoreTestSuite) TestSearchWithPostgres() {
	ctx := sac.WithAllAccess(context.Background())
	img1 := fixtures.GetImageWithUniqueComponents()
	img1.Id = uuid.NewV4().String()
	img2 := fixtures.GetImageWithUniqueComponents()
	img2.Id = uuid.NewV4().String()
	img2.Scan.OperatingSystem = "pluto"
	for _, component := range img2.GetScan().GetComponents() {
		component.Name = img2.Id + component.Name
		for _, vuln := range component.GetVulns() {
			vuln.Cve = img2.Id + vuln.Cve
		}
	}
	img3 := fixtures.GetImageWithUniqueComponents()
	img3.Id = uuid.NewV4().String()

	dep1 := fixtures.GetDeploymentWithImage("c1", "n1", img1)
	dep2 := fixtures.GetDeploymentWithImage("c1", "n2", img2)
	dep3 := fixtures.GetDeploymentWithImage("c2", "n1", img3)

	// Upsert images.
	s.NoError(s.imageDatastore.UpsertImage(ctx, img1))
	s.NoError(s.imageDatastore.UpsertImage(ctx, img2))
	s.NoError(s.imageDatastore.UpsertImage(ctx, img3))
	// Upsert Deployments.
	s.NoError(s.deploymentDatastore.UpsertDeployment(ctx, dep1))
	s.NoError(s.deploymentDatastore.UpsertDeployment(ctx, dep2))
	s.NoError(s.deploymentDatastore.UpsertDeployment(ctx, dep3))

	for _, tc := range []struct {
		desc         string
		ctx          context.Context
		query        *v1.Query
		orderMatters bool
		expectedIDs  []string
		queryImages  bool
	}{
		{
			desc:         "Search deployments with empty query",
			ctx:          ctx,
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.Id, dep2.Id, dep3.Id},
		},
		{
			desc:         "Search deployments with query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.DeploymentID, dep1.Id).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.Id},
		},
		{
			desc:         "Search deployments with image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, img2.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.Id},
		},
		{
			desc:         "Search deployments with non-matching image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, "mars").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search deployments with deployments+image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).AddExactMatches(pkgSearch.ImageOS, img2.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.Id},
		},
		{
			desc:         "Search deployments with deployment scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep1.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.Id},
		},
		{
			desc:         "Search deployments with deployments scope and in-scope deployments query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep1.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep1.Namespace).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.Id},
		},
		{
			desc:         "Search deployments with deployments scope and out-of-scope deployments query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep1.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.Namespace).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search deployments with deployment scope and in-scope image query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep2.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, img2.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.Id},
		},
		{
			desc:         "Search deployments with deployment scope and out-of-scope image query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep2.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.ImageOS, img3.GetScan().GetOperatingSystem()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc:         "Search deployments with image scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: img2.Id, Level: v1.SearchCategory_IMAGES}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.Id},
		},
		{
			desc:         "Search deployments with image scope and in-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: img2.Id, Level: v1.SearchCategory_IMAGES}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.Id},
		},
		{
			desc:         "Search deployments with image scope and out-of-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: img2.Id, Level: v1.SearchCategory_IMAGES}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep3.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
		},
		{
			desc: "Search deployments with image component scope",
			ctx: scoped.Context(ctx, scoped.Scope{
				ID: scancomponent.ComponentID(
					img2.GetScan().GetComponents()[0].GetName(),
					img2.GetScan().GetComponents()[0].GetVersion(),
					img2.GetScan().GetOperatingSystem()),
				Level: v1.SearchCategory_IMAGE_COMPONENTS,
			}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep2.Id},
		},
		{
			desc: "Search deployments with image vuln scope",
			ctx: scoped.Context(ctx, scoped.Scope{
				ID: cve.ID(
					img1.GetScan().GetComponents()[0].GetVulns()[0].GetCve(),
					img1.GetScan().GetOperatingSystem()),
				Level: v1.SearchCategory_IMAGE_VULNERABILITIES,
			}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{dep1.Id, dep3.Id},
		},
		{
			desc:         "Search images with empty query",
			ctx:          ctx,
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1.Id, img2.Id, img3.Id},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: true,
			expectedIDs:  []string{img2.Id},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment+image query",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddExactMatches(pkgSearch.ImageName, img1.GetName().GetFullName()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1.Id, img3.Id},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment+image non-matching search fields",
			ctx:          ctx,
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").AddExactMatches(pkgSearch.ImageSHA, img2.Id).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryImages:  true,
		},
		{
			desc:         "Search images with image scope and in-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: img2.Id, Level: v1.SearchCategory_IMAGES}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep2.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img2.Id},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment scope",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep1.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.EmptyQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1.Id},
			queryImages:  true,
		},
		{
			desc:         "Search images with image scope and out-of-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: img2.Id, Level: v1.SearchCategory_IMAGES}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, dep1.GetNamespace()).ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment scope and in-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep1.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n1").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{img1.Id},
			queryImages:  true,
		},
		{
			desc:         "Search images with deployment scope and out-of-scope deployment query",
			ctx:          scoped.Context(ctx, scoped.Scope{ID: dep1.Id, Level: v1.SearchCategory_DEPLOYMENTS}),
			query:        pkgSearch.NewQueryBuilder().AddExactMatches(pkgSearch.Namespace, "n2").ProtoQuery(),
			orderMatters: false,
			expectedIDs:  []string{},
			queryImages:  true,
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			var actual []pkgSearch.Result
			var err error
			if tc.queryImages {
				actual, err = s.imageDatastore.Search(tc.ctx, tc.query)
			} else {
				actual, err = s.deploymentDatastore.Search(tc.ctx, tc.query)
			}
			assert.NoError(t, err)
			assert.Len(t, actual, len(tc.expectedIDs))
			actualIDs := pkgSearch.ResultsToIDs(actual)
			if tc.orderMatters {
				assert.Equal(t, tc.expectedIDs, actualIDs)
			} else {
				assert.ElementsMatch(t, tc.expectedIDs, actualIDs)
			}
		})
	}
}
