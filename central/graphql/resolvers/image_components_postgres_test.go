//go:build sql_integration

package resolvers

import (
	"context"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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
	testDB   *pgtest.TestPostgres
	resolver *Resolver
}

func (s *GraphQLImageComponentTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = SetupTestPostgresConn(s.T())
	imageDataStore := CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
	resolver, _ := SetupTestResolver(s.T(),
		imageDataStore,
		CreateTestImageComponentDatastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageComponentEdgeDatastore(s.T(), s.testDB),
		CreateTestImageCVEDatastore(s.T(), s.testDB),
		CreateTestImageComponentCVEEdgeDatastore(s.T(), s.testDB),
		CreateTestImageCVEEdgeDatastore(s.T(), s.testDB),
		CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imageDataStore),
	)
	s.resolver = resolver

	// Add Test Data to DataStores
	testDeployments := testDeployments()
	for _, dep := range testDeployments {
		err := s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}

	testImages := testImagesWithOperatingSystems()
	for _, image := range testImages {
		err := s.resolver.ImageDataStore.UpsertImage(s.ctx, image)
		s.NoError(err)
	}
}

func (s *GraphQLImageComponentTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
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

	for _, component := range comps {
		verifyLocationAndLayerIndex(ctx, s.T(), component, true)
	}

	count, err := s.resolver.ImageComponentCount(ctx, RawQuery{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedCount, count)
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentsScoped() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                 string
		id                   string
		expectedComponentIDs []string
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
			expectedCount := int32(len(test.expectedComponentIDs))

			components, err := image.ImageComponents(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(components)))
			assert.ElementsMatch(t, test.expectedComponentIDs, getIDList(ctx, components))

			for _, component := range components {
				verifyLocationAndLayerIndex(ctx, s.T(), component, false)
			}

			count, err := image.ImageComponentCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentsScopeTree() {
	if !features.VulnMgmtWorkloadCVEs.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.VulnMgmtWorkloadCVEs.EnvVar())
		s.T().SkipNow()
	}

	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                      string
		id                        string
		cveToExpectedComponentIDs map[string][]string
	}{
		{
			"sha1",
			"sha1",
			map[string][]string{
				"cve-2018-1": {
					scancomponent.ComponentID("comp1", "0.9", "os1"),
					scancomponent.ComponentID("comp2", "1.1", "os1"),
				},
				"cve-2019-1": {
					scancomponent.ComponentID("comp3", "1.0", "os1"),
				},
				"cve-2019-2": {
					scancomponent.ComponentID("comp3", "1.0", "os1"),
				},
			},
		},
		{
			"sha2",
			"sha2",
			map[string][]string{
				"cve-2018-1": {
					scancomponent.ComponentID("comp1", "0.9", "os2"),
				},
				"cve-2019-1": {
					scancomponent.ComponentID("comp3", "1.0", "os2"),
				},
				"cve-2019-2": {
					scancomponent.ComponentID("comp3", "1.0", "os2"),
				},
				"cve-2017-1": {
					scancomponent.ComponentID("comp4", "1.0", "os2"),
				},
				"cve-2017-2": {
					scancomponent.ComponentID("comp4", "1.0", "os2"),
				},
			},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			image := s.getImageResolver(ctx, test.id)

			vulns, err := image.ImageVulnerabilities(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			for _, vuln := range vulns {
				components, err := vuln.ImageComponents(ctx, PaginatedQuery{})
				assert.NoError(t, err)
				expectedComponents := test.cveToExpectedComponentIDs[vuln.CVE(ctx)]
				require.NotNil(t, expectedComponents)

				expectedCount := int32(len(expectedComponents))
				assert.Equal(t, expectedCount, int32(len(components)))
				assert.ElementsMatch(t, expectedComponents, getIDList(ctx, components))

				for _, component := range components {
					verifyLocationAndLayerIndex(ctx, t, component, false)
				}

				count, err := vuln.ImageComponentCount(ctx, RawQuery{})
				assert.NoError(t, err)
				assert.Equal(t, expectedCount, count)
			}
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
		name                 string
		id                   string
		expectedComponentIDs []string
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
			expectedCount := int32(len(test.expectedComponentIDs))

			images, err := comp.Images(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(images)))
			assert.ElementsMatch(t, test.expectedComponentIDs, getIDList(ctx, images))

			count, err := comp.ImageCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentTestSuite) TestImageComponentImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                 string
		id                   string
		expectedComponentIDs []string
		expectedCounter      *VulnerabilityCounterResolver
	}{
		{
			"comp1os1",
			scancomponent.ComponentID("comp1", "0.9", "os1"),
			[]string{
				cve.ID("cve-2018-1", "os1"),
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
			"comp2os1",
			scancomponent.ComponentID("comp2", "1.1", "os1"),
			[]string{
				cve.ID("cve-2018-1", "os1"),
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
				moderate:  &VulnerabilityFixableCounterResolver{1, 0},
				low:       &VulnerabilityFixableCounterResolver{1, 0},
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
			expectedCount := int32(len(test.expectedComponentIDs))
			test.expectedCounter.all.total = expectedCount

			vulns, err := comp.ImageVulnerabilities(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(vulns)))
			assert.ElementsMatch(t, test.expectedComponentIDs, getIDList(ctx, vulns))

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
		name                 string
		id                   string
		expectedComponentIDs []string
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
			expectedCount := int32(len(test.expectedComponentIDs))

			deps, err := comp.Deployments(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(deps)))
			assert.ElementsMatch(t, test.expectedComponentIDs, getIDList(ctx, deps))

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

func verifyLocationAndLayerIndex(ctx context.Context, t *testing.T, component ImageComponentResolver, assertEmpty bool) {
	if strings.EqualFold(component.Source(ctx), storage.SourceType_OS.String()) {
		return
	}

	if assertEmpty {
		loc, err := component.Location(ctx, RawQuery{})
		assert.NoError(t, err)
		assert.Empty(t, loc)

		layerIdx, err := component.LayerIndex()
		assert.NoError(t, err)
		assert.Zero(t, layerIdx)
		return
	}

	loc, err := component.Location(ctx, RawQuery{})
	assert.NoError(t, err)
	assert.NotEmpty(t, loc)

	layerIdx, err := component.LayerIndex()
	assert.NoError(t, err)
	assert.NotZero(t, layerIdx)
}
