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
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
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
}

func (s *GraphQLImageComponentTestSuite) SetupSuite() {
	s.T().Setenv(env.PostgresDatastoreEnabled.EnvVar(), "true")

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
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
	s.resolver.DeploymentDataStore, err = getDeploymentDatastore(s.ctx, s.T(), s.db, s.gormDB, s.resolver.ImageDataStore, riskMock)
	s.NoError(err, "Failed to get DeploymentDataStore")

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

	testImages := testImagesWithOperatingSystems()
	for _, image := range testImages {
		err = s.resolver.ImageDataStore.UpsertImage(s.ctx, image)
		s.NoError(err)
	}
}

func (s *GraphQLImageComponentTestSuite) TearDownSuite() {

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
	assert.Error(s.T(), err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentTestSuite) TestUnauthorizedImageComponentsEndpoint() {
	_, err := s.resolver.ImageComponents(s.ctx, PaginatedQuery{})
	assert.Error(s.T(), err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentTestSuite) TestUnauthorizedImageComponentCountEndpoint() {
	_, err := s.resolver.ImageComponentCount(s.ctx, RawQuery{})
	assert.Error(s.T(), err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentTestSuite) TestImageComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expectedIDs := []string{
		scancomponent.ComponentID("comp1", "0.9", "os1"),
		scancomponent.ComponentID("comp1", "0.9", "os2"),
		scancomponent.ComponentID("comp2", "1.1", "os1"),
		scancomponent.ComponentID("comp3", "1.0", "os1"),
		scancomponent.ComponentID("comp3", "1.0", "os2"),
		scancomponent.ComponentID("comp4", "1.0", "os2"),
	}
	expectedCount := int32(len(expectedIDs))

	comps, err := s.resolver.ImageComponents(ctx, PaginatedQuery{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedCount, int32(len(comps)))
	assert.ElementsMatch(s.T(), expectedIDs, getIDList(ctx, comps))

	count, err := s.resolver.ImageComponentCount(ctx, RawQuery{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedCount, count)
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
			[]string{
				scancomponent.ComponentID("comp1", "0.9", "os1"),
				scancomponent.ComponentID("comp2", "1.1", "os1"),
				scancomponent.ComponentID("comp3", "1.0", "os1"),
			},
		},
		{
			"sha2",
			"sha2",
			[]string{
				scancomponent.ComponentID("comp1", "0.9", "os2"),
				scancomponent.ComponentID("comp3", "1.0", "os2"),
				scancomponent.ComponentID("comp4", "1.0", "os2"),
			},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			image := s.getImageResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			comps, err := image.ImageComponents(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(comps)))
			assert.ElementsMatch(t, test.expectedIDs, getIDList(ctx, comps))

			count, err := image.ImageComponentCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID("invalid")

	_, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	assert.Error(s.T(), err)
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID(scancomponent.ComponentID("comp1", "0.9", "os1"))

	comp, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), compID, comp.Id(ctx))
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name        string
		id          string
		expectedIDs []string
	}{
		{
			"comp1os1",
			scancomponent.ComponentID("comp1", "0.9", "os1"),
			[]string{"sha1"},
		},
		{
			"comp1os2",
			scancomponent.ComponentID("comp1", "0.9", "os2"),
			[]string{"sha2"},
		},
		{
			"comp2os1",
			scancomponent.ComponentID("comp2", "1.1", "os1"),
			[]string{"sha1"},
		},
		{
			"comp3os1",
			scancomponent.ComponentID("comp3", "1.0", "os1"),
			[]string{"sha1"},
		},
		{
			"comp3os2",
			scancomponent.ComponentID("comp3", "1.0", "os2"),
			[]string{"sha2"},
		},
		{
			"comp4os2",
			scancomponent.ComponentID("comp4", "1.0", "os2"),
			[]string{"sha2"},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			images, err := comp.Images(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(images)))
			assert.ElementsMatch(t, test.expectedIDs, getIDList(ctx, images))

			count, err := comp.ImageCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name            string
		id              string
		expectedIDs     []string
		expectedCounter *VulnerabilityCounterResolver
	}{
		{
			"comp1os1",
			scancomponent.ComponentID("comp1", "0.9", "os1"),
			[]string{
				cve.ID("cve-2018-1", "os1"),
			},
			&VulnerabilityCounterResolver{
				all:       &VulnerabilityFixableCounterResolver{0, 1},
				critical:  &VulnerabilityFixableCounterResolver{0, 0},
				important: &VulnerabilityFixableCounterResolver{0, 0},
				moderate:  &VulnerabilityFixableCounterResolver{0, 0},
				low:       &VulnerabilityFixableCounterResolver{1, 1},
			},
		},
		{
			"comp2os1",
			scancomponent.ComponentID("comp2", "1.1", "os1"),
			[]string{
				cve.ID("cve-2018-1", "os1"),
			},
			&VulnerabilityCounterResolver{
				all:       &VulnerabilityFixableCounterResolver{0, 1},
				critical:  &VulnerabilityFixableCounterResolver{0, 0},
				important: &VulnerabilityFixableCounterResolver{0, 0},
				moderate:  &VulnerabilityFixableCounterResolver{0, 0},
				low:       &VulnerabilityFixableCounterResolver{1, 1},
			},
		},
		{
			"comp3os1",
			scancomponent.ComponentID("comp3", "1.0", "os1"),
			[]string{
				cve.ID("cve-2019-1", "os1"),
				cve.ID("cve-2019-2", "os1"),
			},
			&VulnerabilityCounterResolver{
				all:       &VulnerabilityFixableCounterResolver{0, 0},
				critical:  &VulnerabilityFixableCounterResolver{0, 0},
				important: &VulnerabilityFixableCounterResolver{0, 0},
				moderate:  &VulnerabilityFixableCounterResolver{0, 0},
				low:       &VulnerabilityFixableCounterResolver{2, 0},
			},
		},
		{
			"comp1os2",
			scancomponent.ComponentID("comp1", "0.9", "os2"),
			[]string{
				cve.ID("cve-2018-1", "os2"),
			},
			&VulnerabilityCounterResolver{
				all:       &VulnerabilityFixableCounterResolver{0, 1},
				critical:  &VulnerabilityFixableCounterResolver{1, 1},
				important: &VulnerabilityFixableCounterResolver{0, 0},
				moderate:  &VulnerabilityFixableCounterResolver{0, 0},
				low:       &VulnerabilityFixableCounterResolver{0, 0},
			},
		},
		{
			"comp3os2",
			scancomponent.ComponentID("comp3", "1.0", "os2"),
			[]string{
				cve.ID("cve-2019-1", "os2"),
				cve.ID("cve-2019-2", "os2"),
			},
			&VulnerabilityCounterResolver{
				all:       &VulnerabilityFixableCounterResolver{0, 0},
				critical:  &VulnerabilityFixableCounterResolver{0, 0},
				important: &VulnerabilityFixableCounterResolver{0, 0},
				moderate:  &VulnerabilityFixableCounterResolver{1, 0},
				low:       &VulnerabilityFixableCounterResolver{1, 0},
			},
		},
		{
			"comp4os2",
			scancomponent.ComponentID("comp4", "1.0", "os2"),
			[]string{
				cve.ID("cve-2017-1", "os2"),
				cve.ID("cve-2017-2", "os2"),
			},
			&VulnerabilityCounterResolver{
				all:       &VulnerabilityFixableCounterResolver{0, 0},
				critical:  &VulnerabilityFixableCounterResolver{0, 0},
				important: &VulnerabilityFixableCounterResolver{2, 0},
				moderate:  &VulnerabilityFixableCounterResolver{0, 0},
				low:       &VulnerabilityFixableCounterResolver{0, 0},
			},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))
			test.expectedCounter.all.total = expectedCount

			vulns, err := comp.ImageVulnerabilities(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(vulns)))
			assert.ElementsMatch(t, test.expectedIDs, getIDList(ctx, vulns))

			count, err := comp.ImageVulnerabilityCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)

			counter, err := comp.ImageVulnerabilityCounter(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, test.expectedCounter, counter)
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
			"comp1os1",
			scancomponent.ComponentID("comp1", "0.9", "os1"),
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp2os1",
			scancomponent.ComponentID("comp2", "1.1", "os1"),
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp3os1",
			scancomponent.ComponentID("comp3", "1.0", "os1"),
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp1os2",
			scancomponent.ComponentID("comp1", "0.9", "os2"),
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
		{
			"comp3os2",
			scancomponent.ComponentID("comp3", "1.0", "os2"),
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
		{
			"comp4os2",
			scancomponent.ComponentID("comp4", "1.0", "os2"),
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedIDs))

			deps, err := comp.Deployments(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(deps)))
			assert.ElementsMatch(t, test.expectedIDs, getIDList(ctx, deps))

			count, err := comp.DeploymentCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestTopImageVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := s.getImageComponentResolver(ctx, scancomponent.ComponentID("comp3", "1.0", "os1"))

	expectedID := graphql.ID(cve.ID("cve-2019-1", "os1"))

	vuln, err := comp.TopImageVulnerability(ctx)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedID, vuln.Id(ctx))
}

func (s *GraphQLImageComponentTestSuite) getImageResolver(ctx context.Context, id string) *imageResolver {
	imageID := graphql.ID(id)

	image, err := s.resolver.Image(ctx, struct{ ID graphql.ID }{ID: imageID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), imageID, image.Id(ctx))
	return image
}

func (s *GraphQLImageComponentTestSuite) getImageComponentResolver(ctx context.Context, id string) ImageComponentResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &vulnID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), vulnID, vuln.Id(ctx))
	return vuln
}
