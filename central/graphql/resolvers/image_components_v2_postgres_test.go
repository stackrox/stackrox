//go:build sql_integration

package resolvers

import (
	"context"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
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

func TestGraphQLImageComponentV2Endpoints(t *testing.T) {
	suite.Run(t, new(GraphQLImageComponentV2TestSuite))
}

/*
Remaining TODO tasks:
- as sub resolver
	- from clusters
	- from namespace
- sub resolvers
	- LastScanned
	- LayerIndex
	- Location
*/

type GraphQLImageComponentV2TestSuite struct {
	suite.Suite

	ctx            context.Context
	testDB         *pgtest.TestPostgres
	resolver       *Resolver
	testImages     []*storage.Image
	componentIDMap map[string]string
}

func (s *GraphQLImageComponentV2TestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = pgtest.ForT(s.T())
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	var resolver *Resolver
	if features.FlattenImageData.Enabled() {
		imgV2DataStore := CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
		resolver, _ = SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			imgV2DataStore,
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			CreateTestDeploymentDatastoreWithImageV2(s.T(), s.testDB, mockCtrl, imgV2DataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	} else {
		imageDataStore := CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
		resolver, _ = SetupTestResolver(s.T(),
			imagesView.NewImageView(s.testDB.DB),
			imageDataStore,
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imageDataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	}
	s.resolver = resolver

	// Add Test Data to DataStores
	testDeployments := testDeployments()
	for _, dep := range testDeployments {
		err := s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, dep)
		s.NoError(err)
	}

	s.testImages = testImagesWithOperatingSystems()
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		for _, image := range s.testImages {
			err := s.resolver.ImageV2DataStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(image))
			s.NoError(err)
		}
	} else {
		for _, image := range s.testImages {
			err := s.resolver.ImageDataStore.UpsertImage(s.ctx, image)
			s.NoError(err)
		}
	}

	s.componentIDMap = s.getComponentIDMap(s.testImages)
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
		s.componentIDMap[comp11],
		s.componentIDMap[comp12],
		s.componentIDMap[comp21],
		s.componentIDMap[comp31],
		s.componentIDMap[comp32],
		s.componentIDMap[comp42],
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
	inBaseImageLayerMap := map[string]bool{
		s.componentIDMap[comp11]: false,
		s.componentIDMap[comp12]: true,
		s.componentIDMap[comp21]: false,
		s.componentIDMap[comp31]: false,
		s.componentIDMap[comp32]: true,
		s.componentIDMap[comp42]: true,
	}
	comps, err := s.resolver.ImageComponents(ctx, PaginatedQuery{})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedCount, int32(len(comps)))
	assert.ElementsMatch(s.T(), expectedIDs, getIDList(ctx, comps))

	for _, component := range comps {
		verifyLocationAndLayerIndex(ctx, s.T(), component, emptyLocationMap[string(component.Id(ctx))])
		expectedInBaseImage := inBaseImageLayerMap[string(component.Id(ctx))]
		assert.Equal(s.T(), expectedInBaseImage, component.InBaseImageLayer(ctx))
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
			"image1",
			s.getImageID(s.testImages[0]),
			[]string{
				s.componentIDMap[comp11],
				s.componentIDMap[comp21],
				s.componentIDMap[comp31],
			},
		},
		{
			"image2",
			s.getImageID(s.testImages[1]),
			[]string{
				s.componentIDMap[comp12],
				s.componentIDMap[comp32],
				s.componentIDMap[comp42],
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
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name                      string
		id                        string
		cveToExpectedComponentIDs map[string][]string
	}{
		{
			"image1",
			s.getImageID(s.testImages[0]),
			map[string][]string{
				"cve-2018-1": {
					s.componentIDMap[comp11],
					s.componentIDMap[comp21],
				},
				"cve-2019-1": {
					s.componentIDMap[comp31],
				},
				"cve-2019-2": {
					s.componentIDMap[comp31],
				},
			},
		},
		{
			"image2",
			s.getImageID(s.testImages[1]),
			map[string][]string{
				"cve-2018-1": {
					s.componentIDMap[comp12],
				},
				"cve-2019-1": {
					s.componentIDMap[comp32],
				},
				"cve-2019-2": {
					s.componentIDMap[comp32],
				},
				"cve-2017-1": {
					s.componentIDMap[comp42],
				},
				"cve-2017-2": {
					s.componentIDMap[comp42],
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

	compID := graphql.ID(s.componentIDMap[comp11])

	comp, err := s.resolver.ImageComponent(ctx, IDQuery{ID: &compID})
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), compID, comp.Id(ctx))
}

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentImages() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageID1 := s.getImageID(s.testImages[0])
	imageID2 := s.getImageID(s.testImages[1])

	imageCompTests := []struct {
		name             string
		id               string
		expectedImageIDs []string
	}{
		{
			"comp1image1",
			s.componentIDMap[comp11],
			[]string{imageID1},
		},
		{
			"comp1image2",
			s.componentIDMap[comp12],
			[]string{imageID2},
		},
		{
			"comp2image1",
			s.componentIDMap[comp21],
			[]string{imageID1},
		},
		{
			"comp3image1",
			s.componentIDMap[comp31],
			[]string{imageID1},
		},
		{
			"comp3image2",
			s.componentIDMap[comp32],
			[]string{imageID2},
		},
		{
			"comp4image2",
			s.componentIDMap[comp42],
			[]string{imageID2},
		},
	}

	for _, test := range imageCompTests {
		s.T().Run(test.name, func(t *testing.T) {
			comp := s.getImageComponentResolver(ctx, test.id)
			expectedCount := int32(len(test.expectedImageIDs))

			images, err := comp.Images(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(images)))
			assert.ElementsMatch(t, test.expectedImageIDs, getIDList(ctx, images))

			count, err := comp.ImageCount(ctx, RawQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, count)
		})
	}
}

func (s *GraphQLImageComponentV2TestSuite) TestImageComponentImageVulnerabilities() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())

	imageCompTests := []struct {
		name            string
		id              string
		expectedCVEIDs  []string
		expectedCounter *VulnerabilityCounterResolver
	}{
		{
			"comp1os1",
			s.componentIDMap[comp11],
			[]string{
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2018-1",
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "1.1",
					},
					Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp11], 0),
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
			s.componentIDMap[comp21],
			[]string{
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2018-1",
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "1.5",
					},
					Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp21], 0),
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
			s.componentIDMap[comp31],
			[]string{
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2019-1",
					Cvss:     4,
					Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp31], 0),
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2019-2",
					Cvss:     3,
					Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp31], 1),
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
			s.componentIDMap[comp12],
			[]string{
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2018-1",
					SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
						FixedBy: "1.1",
					},
					Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp12], 0),
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
			s.componentIDMap[comp32],
			[]string{
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2019-1",
					Cvss:     4,
					Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp32], 0),
				getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2019-2",
					Cvss:     3,
					Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp32], 1),
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
			s.componentIDMap[comp42],
			[]string{
				getTestCVEID(&storage.EmbeddedVulnerability{
					Cve:      "cve-2017-1",
					Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp42], 0),
				getTestCVEID(&storage.EmbeddedVulnerability{
					Cve:      "cve-2017-2",
					Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
				}, s.componentIDMap[comp42], 1),
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
			expectedCount := int32(len(test.expectedCVEIDs))
			test.expectedCounter.all.total = expectedCount

			vulns, err := comp.ImageVulnerabilities(ctx, PaginatedQuery{})
			assert.NoError(t, err)
			assert.Equal(t, expectedCount, int32(len(vulns)))
			assert.ElementsMatch(t, test.expectedCVEIDs, getIDList(ctx, vulns))

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
			s.componentIDMap[comp11],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp2os1",
			s.componentIDMap[comp21],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp3os1",
			s.componentIDMap[comp31],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment2},
		},
		{
			"comp1os2",
			s.componentIDMap[comp12],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
		{
			"comp3os2",
			s.componentIDMap[comp32],
			[]string{fixtureconsts.Deployment1, fixtureconsts.Deployment3},
		},
		{
			"comp4os2",
			s.componentIDMap[comp42],
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

	comp := s.getImageComponentResolver(ctx, s.componentIDMap[comp31])

	expectedID := graphql.ID(getTestCVEID(&storage.EmbeddedVulnerability{Cve: "cve-2019-1",
		Cvss:     4,
		Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
	}, s.componentIDMap[comp31], 0))

	vuln, err := comp.TopImageVulnerability(ctx)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), expectedID, vuln.Id(ctx))
}

func (s *GraphQLImageComponentV2TestSuite) getImageResolver(ctx context.Context, id string) ImageResolver {
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

// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
func (s *GraphQLImageComponentV2TestSuite) getImageID(image *storage.Image) string {
	if features.FlattenImageData.Enabled() {
		return imageUtils.ConvertToV2(image).GetId()
	}
	return image.GetId()
}

func (s *GraphQLImageComponentV2TestSuite) getComponentIDMap(images []*storage.Image) map[string]string {
	imageID1 := s.getImageID(images[0])
	imageID2 := s.getImageID(images[1])
	return map[string]string{
		comp11: getTestComponentID(images[0].GetScan().GetComponents()[0], imageID1, 0),
		comp12: getTestComponentID(images[1].GetScan().GetComponents()[0], imageID2, 0),
		comp21: getTestComponentID(images[0].GetScan().GetComponents()[1], imageID1, 1),
		comp31: getTestComponentID(images[0].GetScan().GetComponents()[2], imageID1, 2),
		comp32: getTestComponentID(images[1].GetScan().GetComponents()[1], imageID2, 1),
		comp42: getTestComponentID(images[1].GetScan().GetComponents()[2], imageID2, 2),
	}
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
