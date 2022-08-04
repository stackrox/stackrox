//go:build sql_integration
// +build sql_integration

package resolvers

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	imageCVESearch "github.com/stackrox/rox/central/cve/image/datastore/search"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentDataStore "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageComponentSearch "github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestGraphQLImageVulnerabilityEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLImageVulnerabilityTestSuite))
}

type GraphQLImageVulnerabilityTestSuite struct {
	suite.Suite

	ctx      context.Context
	db       *pgxpool.Pool
	gormDB   *gorm.DB
	resolver *Resolver

	envIsolator *envisolator.EnvIsolator
}

func (s *GraphQLImageVulnerabilityTestSuite) SetupSuite() {

	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(features.PostgresDatastore.EnvVar(), "true")

	if !features.PostgresDatastore.Enabled() {
		s.T().Skip("Skip postgres store tests")
		s.T().SkipNow()
	}

	s.ctx = context.Background()

	source := pgtest.GetConnectionString(s.T())
	config, err := pgxpool.ParseConfig(source)
	s.NoError(err)

	pool, err := pgxpool.ConnectConfig(s.ctx, config)
	s.NoError(err)
	s.gormDB = pgtest.OpenGormDB(s.T(), source)
	s.db = pool

	// destroy datastores if they exist
	imagePostgres.Destroy(s.ctx, s.db)
	imageComponentPostgres.Destroy(s.ctx, s.db)
	imageCVEPostgres.Destroy(s.ctx, s.db)

	// create mock resolvers, set relevant ones
	s.resolver = NewMock()

	// imageCVE datastore
	imageCVEStore := imageCVEPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	imageCVEIndexer := imageCVEPostgres.NewIndexer(s.db)
	imageCVESearcher := imageCVESearch.New(imageCVEStore, imageCVEIndexer)
	imageCVEDatastore, err := imageCVEDataStore.New(imageCVEStore, imageCVEIndexer, imageCVESearcher, concurrency.NewKeyFence())
	s.NoError(err, "Failed to create ImageCVEDatastore")
	s.resolver.ImageCVEDataStore = imageCVEDatastore

	// image datastore
	riskMock := mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	imageStore := imagePostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB, false)
	s.resolver.ImageDataStore = imageDataStore.NewWithPostgres(imageStore, imagePostgres.NewIndexer(s.db), riskMock, ranking.NewRanker(), ranking.NewRanker())

	// image component datastore
	imageCompStore := imageComponentPostgres.CreateTableAndNewStore(s.ctx, s.db, s.gormDB)
	imageCompIndexer := imageComponentPostgres.NewIndexer(s.db)
	imageCompSearcher := imageComponentSearch.NewV2(imageCompStore, imageCompIndexer)
	s.resolver.ImageComponentDataStore = imageComponentDataStore.New(nil, imageCompStore, imageCompIndexer, imageCompSearcher, riskMock, ranking.NewRanker())

	// Sac permissions
	s.ctx = sac.WithAllAccess(s.ctx)

	// loaders used by graphql layer
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.Image{}), func() interface{} {
		return loaders.NewImageLoader(s.resolver.ImageDataStore)
	})
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageComponent{}), func() interface{} {
		return loaders.NewComponentLoader(s.resolver.ImageComponentDataStore)
	})
	loaders.RegisterTypeFactory(reflect.TypeOf(storage.ImageCVE{}), func() interface{} {
		return loaders.NewImageCVELoader(s.resolver.ImageCVEDataStore)
	})
	s.ctx = loaders.WithLoaderContext(s.ctx)

	// Add Test Data to DataStores
	testImages := testImages()
	for _, image := range testImages {
		err = imageStore.Upsert(s.ctx, image)
		s.NoError(err)
	}
}

func (s *GraphQLImageVulnerabilityTestSuite) TearDownSuite() {
	imagePostgres.Destroy(s.ctx, s.db)
	imageComponentPostgres.Destroy(s.ctx, s.db)
	imageCVEPostgres.Destroy(s.ctx, s.db)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilityEndpoint() {
	_, err := s.resolver.ImageVulnerability(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilitiesEndpoint() {
	_, err := s.resolver.ImageVulnerabilities(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilityCountEndpoint() {
	_, err := s.resolver.ImageVulnerabilityCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedImageVulnerabilityCounterEndpoint() {
	_, err := s.resolver.ImageVulnerabilityCounter(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestUnauthorizedTopImageVulnerabilityEndpoint() {
	_, err := s.resolver.TopImageVulnerability(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(5, len(vulns))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	query, err := getFixableRawQuery(true)
	s.NoError(err)

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(1, len(vulns))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesNonFixable() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	query, err := getFixableRawQuery(false)
	s.NoError(err)

	vulns, err := s.resolver.ImageVulnerabilities(ctx, PaginatedQuery{Query: &query})
	s.NoError(err)
	s.Equal(4, len(vulns))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilitiesScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")

	vulns, err := image.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(3, len(vulns))

	image = s.getImageResolver(ctx, "sha2")

	vulns, err = image.ImageVulnerabilities(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(5, len(vulns))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("invalid")

	_, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.Error(err)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vulnID := graphql.ID("cve-2018-1#")

	vuln, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.ID(ctx))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityCount() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	count, err := s.resolver.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(5), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityCountScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")

	count, err := image.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(3), count)

	image = s.getImageResolver(ctx, "sha2")

	count, err = image.ImageVulnerabilityCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(5), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityCounter() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	count, err := s.resolver.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), count, 5, 1, 1, 2, 1, 1)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityCounterScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")

	count, err := image.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), count, 3, 1, 1, 0, 1, 1)

	image = s.getImageResolver(ctx, "sha2")

	count, err = image.ImageVulnerabilityCounter(ctx, RawQuery{})
	s.NoError(err)
	checkVulnerabilityCounter(s.T(), count, 5, 1, 1, 2, 1, 1)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestTopImageVulnerabilityUnscoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	_, err := s.resolver.TopImageVulnerability(ctx, RawQuery{})
	s.Error(err)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestTopImageVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	image := s.getImageResolver(ctx, "sha1")

	_, err := image.TopImageVulnerability(ctx, RawQuery{})
	s.NoError(err)

	// TODO figure out how to test this
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")

	images, err := vuln.Images(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(images))

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")

	images, err = vuln.Images(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(images))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityImageCount() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")

	count, err := vuln.ImageCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(2), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")

	count, err = vuln.ImageCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityImageComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")

	images, err := vuln.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(2, len(images))

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")

	images, err = vuln.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(1, len(images))
}

func (s *GraphQLImageVulnerabilityTestSuite) TestImageVulnerabilityImageComponentCount() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	vuln := s.getImageVulnerabilityResolver(ctx, "cve-2018-1#")

	count, err := vuln.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(2), count)

	vuln = s.getImageVulnerabilityResolver(ctx, "cve-2017-1#")

	count, err = vuln.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(int32(1), count)
}

func (s *GraphQLImageVulnerabilityTestSuite) getImageResolver(ctx context.Context, id string) *imageResolver {
	imageID := graphql.ID(id)

	image, err := s.resolver.Image(ctx, struct{ ID graphql.ID }{ID: imageID})
	s.NoError(err)
	s.Equal(imageID, image.Id(ctx))
	return image
}

func (s *GraphQLImageVulnerabilityTestSuite) getImageVulnerabilityResolver(ctx context.Context, id string) ImageVulnerabilityResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ImageVulnerability(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.ID(ctx))
	return vuln
}
