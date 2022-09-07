//go:build sql_integration

package resolvers

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"
	componentCVEEdgePostgres "github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	imageCVEPostgres "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	deploymentPostgres "github.com/stackrox/rox/central/deployment/store/postgres"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imagePostgres "github.com/stackrox/rox/central/image/datastore/store/postgres"
	imageComponentPostgres "github.com/stackrox/rox/central/imagecomponent/datastore/store/postgres"
	imageCVEEdgePostgres "github.com/stackrox/rox/central/imagecveedge/datastore/postgres"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

func TestGraphQLImageComponentEndpoints(t *testing.T) {
	suite.Run(t, new(GraphQLImageComponentTestSuite))
}

/*
Remaining TODO tasks:
- as sub resolver
	- from clusters
	- from namespace
- sub resolvers
	- ActiveState
	- LastScanned
	- LayerIndex
	- Location
*/

type GraphQLImageComponentTestSuite struct {
	suite.Suite

	ctx      context.Context
	db       *pgxpool.Pool
	gormDB   *gorm.DB
	resolver *Resolver

	envIsolator *envisolator.EnvIsolator
}

func (s *GraphQLImageComponentTestSuite) SetupSuite() {

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
	imageCVEPostgres.Destroy(s.ctx, s.db)
	imagePostgres.Destroy(s.ctx, s.db)
	imageComponentPostgres.Destroy(s.ctx, s.db)
	imageCVEEdgePostgres.Destroy(s.ctx, s.db)
	componentCVEEdgePostgres.Destroy(s.ctx, s.db)
	deploymentPostgres.Destroy(s.ctx, s.db)

	// create mock resolvers, set relevant ones
	s.resolver = NewMock()

	riskMock := mockRisks.NewMockDataStore(gomock.NewController(s.T()))

	s.resolver.ImageCVEDataStore, err = getImageCVEDatastore(s.ctx, s.db, s.gormDB)
	s.NoError(err, "Failed to get ImageCVEDataStore")
	s.resolver.ImageDataStore = getImageDatastore(s.ctx, s.db, s.gormDB, riskMock)
	s.resolver.ImageComponentDataStore = getImageComponentDatastore(s.ctx, s.db, s.gormDB, riskMock)
	s.resolver.ImageCVEEdgeDataStore = getImageCVEEdgeDatastore(s.ctx, s.db, s.gormDB)
	s.resolver.ComponentCVEEdgeDataStore = getImageComponentCVEEdgeDatastore(s.ctx, s.db, s.gormDB)
	s.resolver.DeploymentDataStore, err = getDeploymentDatastore(s.ctx, s.db, s.gormDB, s.resolver.ImageDataStore, riskMock)

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
	testDeployments := testDeployments()
	for _, dep := range testDeployments {
		err = s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}

	testImages := testImages()
	for _, image := range testImages {
		err = s.resolver.ImageDataStore.UpsertImage(s.ctx, image)
		s.NoError(err)
	}
}

func (s *GraphQLImageComponentTestSuite) TearDownSuite() {
	s.envIsolator.RestoreAll()

	imageCVEPostgres.Destroy(s.ctx, s.db)
	imagePostgres.Destroy(s.ctx, s.db)
	imageComponentPostgres.Destroy(s.ctx, s.db)
	imageCVEEdgePostgres.Destroy(s.ctx, s.db)
	componentCVEEdgePostgres.Destroy(s.ctx, s.db)
	deploymentPostgres.Destroy(s.ctx, s.db)
	pgtest.CloseGormDB(s.T(), s.gormDB)
	s.db.Close()
}

func (s *GraphQLImageComponentTestSuite) TestUnauthorizedImageComponentEndpoint() {
	_, err := s.resolver.ImageComponent(s.ctx, IDQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentTestSuite) TestUnauthorizedImageComponentsEndpoint() {
	_, err := s.resolver.ImageComponents(s.ctx, PaginatedQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentTestSuite) TestUnauthorizedImageComponentCountEndpoint() {
	_, err := s.resolver.ImageComponentCount(s.ctx, RawQuery{})
	s.Error(err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentTestSuite) TestImageComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expectedIds := []string{"comp1#0.9#", "comp2#1.1#", "comp3#1.0#", "comp4#1.0#"}
	expectedCount := int32(len(expectedIds))

	comps, err := s.resolver.ImageComponents(ctx, PaginatedQuery{})
	s.NoError(err)
	s.Equal(expectedCount, int32(len(comps)))
	idList := getIDList(ctx, comps)
	s.ElementsMatch(expectedIds, idList)

	count, err := s.resolver.ImageComponentCount(ctx, RawQuery{})
	s.NoError(err)
	s.Equal(expectedCount, count)
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentsScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name        string
		id          string
		expectedIDs []string
	}{
		{
			"sha1",
			"sha1",
			[]string{"comp1#0.9#", "comp2#1.1#", "comp3#1.0#"},
		},
		{
			"sha2",
			"sha2",
			[]string{"comp1#0.9#", "comp3#1.0#", "comp4#1.0#"},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			image := s.getImageResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			comps, err := image.ImageComponents(ctx, PaginatedQuery{})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(comps)))
			idList := getIDList(ctx, comps)
			s.ElementsMatch(test.expectedIDs, idList)

			count, err := image.ImageComponentCount(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID("invalid")

	_, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	s.Error(err)
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID("comp1#0.9#")

	comp, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	s.NoError(err)
	s.Equal(compID, comp.Id(ctx))
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name        string
		id          string
		expectedIDs []string
	}{
		{
			"comp1",
			"comp1#0.9#",
			[]string{"sha1", "sha2"},
		},
		{
			"comp2",
			"comp2#1.1#",
			[]string{"sha1"},
		},
		{
			"comp3",
			"comp3#1.0#",
			[]string{"sha1", "sha2"},
		},
		{
			"comp4",
			"comp4#1.0#",
			[]string{"sha2"},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			images, err := comp.Images(ctx, PaginatedQuery{})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(images)))
			idList := getIDList(ctx, images)
			s.ElementsMatch(test.expectedIDs, idList)

			count, err := comp.ImageCount(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	type counterValues struct {
		fixable   int32
		critical  int32
		important int32
		moderate  int32
		low       int32
	}

	imageCompTests := []struct {
		name                  string
		id                    string
		expectedIDs           []string
		expectedCounterValues counterValues
	}{
		{
			"comp1",
			"comp1#0.9#",
			[]string{"cve-2018-1#"},
			counterValues{
				1, 1, 0, 0, 0,
			},
		},
		{
			"comp2",
			"comp2#1.1#",
			[]string{"cve-2018-1#"},
			counterValues{
				1, 1, 0, 0, 0,
			},
		},
		{
			"comp3",
			"comp3#1.0#",
			[]string{"cve-2019-1#", "cve-2019-2#"},
			counterValues{
				0, 0, 0, 1, 1,
			},
		},
		{
			"comp4",
			"comp4#1.0#",
			[]string{"cve-2017-1#", "cve-2017-2#"},
			counterValues{
				0, 0, 2, 0, 0,
			},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			vulns, err := comp.ImageVulnerabilities(ctx, PaginatedQuery{})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(vulns)))
			idList := getIDList(ctx, vulns)
			s.ElementsMatch(test.expectedIDs, idList)

			count, err := comp.ImageVulnerabilityCount(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, count)

			counter, err := comp.ImageVulnerabilityCounter(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, counter.All(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.fixable, counter.All(ctx).Fixable(ctx))
			s.Equal(test.expectedCounterValues.critical, counter.Critical(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.important, counter.Important(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.moderate, counter.Moderate(ctx).Total(ctx))
			s.Equal(test.expectedCounterValues.low, counter.Low(ctx).Total(ctx))
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentDeployments() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name        string
		id          string
		expectedIDs []string
	}{
		{
			"comp1",
			"comp1#0.9#",
			[]string{"dep1id", "dep2id", "dep3id"},
		},
		{
			"comp2",
			"comp2#1.1#",
			[]string{"dep1id", "dep2id"},
		},
		{
			"comp3",
			"comp3#1.0#",
			[]string{"dep1id", "dep2id", "dep3id"},
		},
		{
			"comp4",
			"comp4#1.0#",
			[]string{"dep1id", "dep3id"},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			deps, err := comp.Deployments(ctx, PaginatedQuery{})
			s.NoError(err)
			s.Equal(expectedCount, int32(len(deps)))
			idList := getIDList(ctx, deps)
			s.ElementsMatch(test.expectedIDs, idList)

			count, err := comp.DeploymentCount(ctx, RawQuery{})
			s.NoError(err)
			s.Equal(expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestTopImageVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := s.getImageComponentResolver(ctx, "comp3#1.0#")

	expectedId := graphql.ID("cve-2019-1#")

	vuln, err := comp.TopImageVulnerability(ctx)
	s.NoError(err)
	s.Equal(expectedId, vuln.Id(ctx))
}

func (s *GraphQLImageComponentTestSuite) getImageResolver(ctx context.Context, id string) *imageResolver {
	imageID := graphql.ID(id)

	image, err := s.resolver.Image(ctx, struct{ ID graphql.ID }{ID: imageID})
	s.NoError(err)
	s.Equal(imageID, image.Id(ctx))
	return image
}

func (s *GraphQLImageComponentTestSuite) getImageComponentResolver(ctx context.Context, id string) ImageComponentResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &vulnID})
	s.NoError(err)
	s.Equal(vulnID, vuln.Id(ctx))
	return vuln
}
