//go:build sql_integration

package resolvers

import (
	"context"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	imagesView "github.com/stackrox/rox/central/views/images"
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

const (
	comp11 = "comp1image1"
	comp12 = "comp1image2"
	comp21 = "comp2image1"
	comp31 = "comp3image1"
	comp32 = "comp3image2"
	comp42 = "comp4image2"
)

var (
	componentIDMap = map[string]string{
		comp11: scancomponent.ComponentIDV2("comp1", "0.9", "", "sha1"),
		comp12: scancomponent.ComponentIDV2("comp1", "0.9", "", "sha2"),
		comp21: scancomponent.ComponentIDV2("comp2", "1.1", "", "sha1"),
		comp31: scancomponent.ComponentIDV2("comp3", "1.0", "", "sha1"),
		comp32: scancomponent.ComponentIDV2("comp3", "1.0", "", "sha2"),
		comp42: scancomponent.ComponentIDV2("comp4", "1.0", "", "sha2"),
	}
)

func TestGraphQLImageComponentV2Endpoints(t *testing.T) {
	suite.Run(t, new(GraphQLImageComponentV2TestSuite))
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

type GraphQLImageComponentV2TestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver
}

func (s *GraphQLImageComponentV2TestSuite) SetupSuite() {
	if !features.FlattenCVEData.Enabled() {
		s.T().Skip()
	}

	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())
	imageDataStore := CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
	resolver, _ := SetupTestResolver(s.T(),
		imagesView.NewImageView(s.testDB.DB),
		imageDataStore,
		CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageCVEV2Datastore(s.T(), s.testDB),
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

func (s *GraphQLImageComponentV2TestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
	s.T().Setenv("ROX_FLATTEN_CVE_DATA", "false")
}

func (s *GraphQLImageComponentV2TestSuite) TestUnauthorizedImageComponentEndpoint() {
	_, err := s.resolver.ImageComponent(s.ctx, IDQuery{})
	assert.Error(s.T(), err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentV2TestSuite) TestUnauthorizedImageComponentsEndpoint() {
	_, err := s.resolver.ImageComponents(s.ctx, PaginatedQuery{})
	assert.Error(s.T(), err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentV2TestSuite) TestUnauthorizedImageComponentCountEndpoint() {
	_, err := s.resolver.ImageComponentCount(s.ctx, RawQuery{})
	assert.Error(s.T(), err, "Unauthorized request got through")
}

func (s *GraphQLImageComponentV2TestSuite) TestImageComponents() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	expectedIDs := []string{
		componentIDMap[comp11],
		componentIDMap[comp12],
		componentIDMap[comp21],
		componentIDMap[comp31],
		componentIDMap[comp32],
		componentIDMap[comp42],
	}
	expectedCount := int32(len(expectedIDs))

	emptyLocationMap := map[string]bool{
		comp11: true,
		comp12: true,
		comp21: true,
		comp31: false,
		comp32: false,
		comp42: true,
	}

	comps, err := s.resolver.ImageComponents(ctx, PaginatedQuery{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedCount, int32(len(comps)))
	assert.ElementsMatch(s.T(), expectedIDs, getIDList(ctx, comps))

	for _, component := range comps {
		verifyLocationAndLayerIndex(ctx, s.T(), component, emptyLocationMap[string(component.Id(ctx))])
	}

	count, err := s.resolver.ImageComponentCount(ctx, RawQuery{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedCount, count)
}

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentsScoped() {
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
				componentIDMap[comp11],
				componentIDMap[comp21],
				componentIDMap[comp31],
			},
		},
		{
			"sha2",
			"sha2",
			[]string{
				componentIDMap[comp12],
				componentIDMap[comp32],
				componentIDMap[comp42],
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

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentsScopeTree() {
	// TODO(ROX-28320): Data here needs to be grouped by CVE not the ID when getting vulns by image
	// as is done in the Image Vulnerabilities resolver.
	s.T().Skip()

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
					componentIDMap[comp11],
					componentIDMap[comp21],
				},
				"cve-2019-1": {
					componentIDMap[comp31],
				},
				"cve-2019-2": {
					componentIDMap[comp31],
				},
			},
		},
		{
			"sha2",
			"sha2",
			map[string][]string{
				"cve-2018-1": {
					componentIDMap[comp12],
				},
				"cve-2019-1": {
					componentIDMap[comp32],
				},
				"cve-2019-2": {
					componentIDMap[comp32],
				},
				"cve-2017-1": {
					componentIDMap[comp42],
				},
				"cve-2017-2": {
					componentIDMap[comp42],
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

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentMiss() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID("invalid")

	_, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	assert.Error(s.T(), err)
}

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentHit() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	compID := graphql.ID(componentIDMap[comp11])

	comp, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), compID, comp.Id(ctx))
}

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                 string
		id                   string
		expectedComponentIDs []string
	}{
		{
			"comp1image1",
			componentIDMap[comp11],
			[]string{"sha1"},
		},
		{
			"comp1image2",
			componentIDMap[comp12],
			[]string{"sha2"},
		},
		{
			"comp2image1",
			componentIDMap[comp21],
			[]string{"sha1"},
		},
		{
			"comp3image1",
			componentIDMap[comp31],
			[]string{"sha1"},
		},
		{
			"comp3image2",
			componentIDMap[comp32],
			[]string{"sha2"},
		},
		{
			"comp4image2",
			componentIDMap[comp42],
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

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                 string
		id                   string
		expectedComponentIDs []string
		expectedCounter      *VulnerabilityCounterResolver
	}{
		{
			"comp1os1",
			componentIDMap[comp11],
			[]string{
				cve.IDV2("cve-2018-1", componentIDMap[comp11], "0"),
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
			componentIDMap[comp21],
			[]string{
				cve.IDV2("cve-2018-1", componentIDMap[comp21], "0"),
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
			componentIDMap[comp31],
			[]string{
				cve.IDV2("cve-2019-1", componentIDMap[comp31], "0"),
				cve.IDV2("cve-2019-2", componentIDMap[comp31], "1"),
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
			componentIDMap[comp12],
			[]string{
				cve.IDV2("cve-2018-1", componentIDMap[comp12], "0"),
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
			componentIDMap[comp32],
			[]string{
				cve.IDV2("cve-2019-1", componentIDMap[comp32], "0"),
				cve.IDV2("cve-2019-2", componentIDMap[comp32], "1"),
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
			componentIDMap[comp42],
			[]string{
				cve.IDV2("cve-2017-1", componentIDMap[comp42], "0"),
				cve.IDV2("cve-2017-2", componentIDMap[comp42], "1"),
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

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentDeployments() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                 string
		id                   string
		expectedComponentIDs []string
	}{
		{
			"comp1os1",
			componentIDMap[comp11],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp2os1",
			componentIDMap[comp21],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp3os1",
			componentIDMap[comp31],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp1os2",
			componentIDMap[comp12],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
		{
			"comp3os2",
			componentIDMap[comp32],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
		{
			"comp4os2",
			componentIDMap[comp42],
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

func (s *GraphQLImageComponentV2TestSuite) TestTopImageVulnerability() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	comp := s.getImageComponentResolver(ctx, componentIDMap[comp31])

	expectedID := graphql.ID(cve.IDV2("cve-2019-1", componentIDMap[comp31], "0"))

	vuln, err := comp.TopImageVulnerability(ctx)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedID, vuln.Id(ctx))
}

func (s *GraphQLImageComponentV2TestSuite) getImageResolver(ctx context.Context, id string) *imageResolver {
	imageID := graphql.ID(id)

	image, err := s.resolver.Image(ctx, struct{ ID graphql.ID }{ID: imageID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), imageID, image.Id(ctx))
	return image
}

func (s *GraphQLImageComponentV2TestSuite) getImageComponentResolver(ctx context.Context, id string) ImageComponentResolver {
	vulnID := graphql.ID(id)

	vuln, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &vulnID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), vulnID, vuln.Id(ctx))
	return vuln
}
