//go:build sql_integration

package resolvers

import (
	"context"
	"encoding/json"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	deploymentsView "github.com/stackrox/rox/central/views/deployments"
	"github.com/stackrox/rox/central/views/imagecomponentflat"
	"github.com/stackrox/rox/central/views/imagecve"
	"github.com/stackrox/rox/central/views/imagecveflat"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageResolvers(t *testing.T) {
	suite.Run(t, new(ImageResolversTestSuite))
}

type ImageResolversTestSuite struct {
	suite.Suite

	ctx      context.Context
	testDB   *pgtest.TestPostgres
	resolver *Resolver
	schema   *graphql.Schema

	testDeployments []*storage.Deployment
	testImages      []*storage.Image
}

func (s *ImageResolversTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = SetupTestPostgresConn(s.T())

	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	var resolver *Resolver
	if features.FlattenImageData.Enabled() {
		imgV2DataStore := CreateTestImageV2Datastore(s.T(), s.testDB, mockCtrl)
		resolver, s.schema = SetupTestResolver(s.T(),
			CreateTestDeploymentDatastoreWithImageV2(s.T(), s.testDB, mockCtrl, imgV2DataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
			imagesView.NewImageView(s.testDB.DB),
			imgV2DataStore,
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			imagecve.NewCVEView(s.testDB.DB),
			imagecveflat.NewCVEFlatView(s.testDB.DB),
			imagecomponentflat.NewComponentFlatView(s.testDB.DB),
		)
	} else {
		imgDataStore := CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
		resolver, s.schema = SetupTestResolver(s.T(),
			CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imgDataStore),
			deploymentsView.NewDeploymentView(s.testDB.DB),
			imagesView.NewImageView(s.testDB.DB),
			imgDataStore,
			CreateTestImageComponentV2Datastore(s.T(), s.testDB, mockCtrl),
			CreateTestImageCVEV2Datastore(s.T(), s.testDB),
			imagecve.NewCVEView(s.testDB.DB),
			imagecveflat.NewCVEFlatView(s.testDB.DB),
		)
	}
	s.resolver = resolver

	// Add Test Data.
	s.testDeployments = append(testDeployments(), occurrenceTestDeployments()...)
	for _, deployment := range s.testDeployments {
		s.NoError(s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, deployment))
	}
	s.testImages = append(testImages(), occurrenceTestImages()...)
	// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
	if features.FlattenImageData.Enabled() {
		for _, image := range s.testImages {
			s.NoError(s.resolver.ImageV2DataStore.UpsertImage(s.ctx, imageUtils.ConvertToV2(image)))
		}
	} else {
		for _, image := range s.testImages {
			s.NoError(s.resolver.ImageDataStore.UpsertImage(s.ctx, image))
		}
	}
}

func sacAllowOnlyCluster2Namespace2(ctx context.Context) context.Context {
	return sac.WithGlobalAccessScopeChecker(
		ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Image, resources.Deployment),
			sac.ClusterScopeKeys(fixtureconsts.Cluster2),
			sac.NamespaceScopeKeys(namespace2name),
		),
	)
}

func (s *ImageResolversTestSuite) TestDeployments() {
	ctx := SetAuthorizerOverride(s.ctx, allow.Anonymous())
	sacCtx := sacAllowOnlyCluster2Namespace2(ctx)

	for _, tc := range []struct {
		desc            string
		ctx             context.Context
		q               PaginatedQuery
		deploymentFiler func(d *storage.Deployment) bool
		imageFilter     func(img *storage.Image) bool
		vulnFilter      func(img *storage.EmbeddedVulnerability) bool
	}{
		{
			desc:            "no filter",
			ctx:             ctx,
			q:               PaginatedQuery{},
			deploymentFiler: func(_ *storage.Deployment) bool { return true },
			imageFilter:     func(_ *storage.Image) bool { return true },
			vulnFilter:      func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc: "filter by namespace",
			ctx:  ctx,
			q:    PaginatedQuery{Query: new("Namespace:namespace1name")},
			deploymentFiler: func(d *storage.Deployment) bool {
				return strings.HasPrefix(d.GetNamespace(), "namespace1name")
			},
			imageFilter: func(_ *storage.Image) bool { return true },
			vulnFilter:  func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc: "filter by deployment",
			ctx:  ctx,
			q:    PaginatedQuery{Query: new("Deployment:" + dep1name)},
			deploymentFiler: func(d *storage.Deployment) bool {
				return d.GetName() == dep1name
			},
			imageFilter: func(_ *storage.Image) bool { return true },
			vulnFilter:  func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc:            "filter by image",
			ctx:             ctx,
			q:               PaginatedQuery{Query: new("Image:reg1/img1")},
			deploymentFiler: func(d *storage.Deployment) bool { return true },
			imageFilter: func(img *storage.Image) bool {
				return strings.HasPrefix(img.GetName().GetFullName(), "reg1/img1")
			},
			vulnFilter: func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc:            "filter by cve",
			ctx:             ctx,
			q:               PaginatedQuery{Query: new("CVE:cve-2019-2")},
			deploymentFiler: func(d *storage.Deployment) bool { return true },
			imageFilter: func(img *storage.Image) bool {
				for _, component := range img.GetScan().GetComponents() {
					for _, vuln := range component.GetVulns() {
						if strings.HasPrefix(vuln.GetCve(), "cve-2019-2") {
							return true
						}
					}
				}
				return false
			},
			vulnFilter: func(v *storage.EmbeddedVulnerability) bool {
				return strings.HasPrefix(v.GetCve(), "cve-2019-2")
			},
		},
		{
			desc: "filter by deployment+cve",
			ctx:  ctx,
			q:    PaginatedQuery{Query: new("Deployment:dep2name+CVE:cve-2019-2")},
			deploymentFiler: func(d *storage.Deployment) bool {
				return strings.HasPrefix(d.GetName(), "dep2name")
			},
			imageFilter: func(img *storage.Image) bool {
				for _, component := range img.GetScan().GetComponents() {
					for _, vuln := range component.GetVulns() {
						if strings.HasPrefix(vuln.GetCve(), "cve-2019-2") {
							return true
						}
					}
				}
				return false
			},
			vulnFilter: func(v *storage.EmbeddedVulnerability) bool {
				return strings.HasPrefix(v.GetCve(), "cve-2019-2")
			},
		},
		{
			desc:            "filter by severity",
			ctx:             ctx,
			q:               PaginatedQuery{Query: new("Severity:CRITICAL_VULNERABILITY_SEVERITY")},
			deploymentFiler: func(d *storage.Deployment) bool { return true },
			imageFilter: func(img *storage.Image) bool {
				for _, component := range img.GetScan().GetComponents() {
					for _, vuln := range component.GetVulns() {
						if strings.HasPrefix(vuln.GetSeverity().String(), "CRITICAL_VULNERABILITY_SEVERITY") {
							return true
						}
					}
				}
				return false
			},
			vulnFilter: func(v *storage.EmbeddedVulnerability) bool {
				return strings.HasPrefix(v.GetSeverity().String(), "CRITICAL_VULNERABILITY_SEVERITY")
			},
		},
		{
			desc:            "filter by severity+fixable",
			ctx:             ctx,
			q:               PaginatedQuery{Query: new("Severity:UNSET_VULNERABILITY_SEVERITY+Fixable:true")},
			deploymentFiler: func(d *storage.Deployment) bool { return true },
			imageFilter: func(img *storage.Image) bool {
				for _, component := range img.GetScan().GetComponents() {
					for _, vuln := range component.GetVulns() {
						if strings.HasPrefix(vuln.GetSeverity().String(), "UNSET_VULNERABILITY_SEVERITY") &&
							vuln.GetFixedBy() != "" {
							return true
						}
					}
				}
				return false
			},
			vulnFilter: func(v *storage.EmbeddedVulnerability) bool {
				return strings.HasPrefix(v.GetSeverity().String(), "CRITICAL_VULNERABILITY_SEVERITY") && v.GetFixedBy() != ""
			},
		},
		{
			desc: "sac filter namespace 2",
			ctx:  sacCtx,
			q:    PaginatedQuery{},
			deploymentFiler: func(d *storage.Deployment) bool {
				return d.GetClusterId() == fixtureconsts.Cluster2 && d.GetNamespace() == "namespace2name"
			},
			imageFilter: func(_ *storage.Image) bool { return true },
			vulnFilter:  func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
	} {
		s.T().Run(tc.desc, func(t *testing.T) {
			paginatedQ := tc.q
			testCtx := tc.ctx

			// Test ImageCount query.
			expectedImages, expectedDeploymentsPerImage := compileExpectedForImageGraphQL(s.testDeployments, s.testImages, tc.deploymentFiler, tc.imageFilter)
			count, err := s.resolver.ImageCount(testCtx, RawQuery{Query: paginatedQ.Query})
			assert.NoError(t, err)
			assert.Equal(t, int32(len(expectedImages)), count)

			// Test Images query.
			actualImages, err := s.resolver.Images(testCtx, paginatedQ)
			require.NoError(t, err)
			var expectedIDs []string
			for imgID := range expectedImages {
				expectedIDs = append(expectedIDs, imgID)
			}
			assert.ElementsMatch(t, expectedIDs, getIDList(testCtx, actualImages))

			for _, image := range actualImages {
				imageID := string(image.Id(testCtx))
				// Test DeploymentCount field for each image resolver.
				expectedDeployments := expectedDeploymentsPerImage[imageID]
				actualDeploymentCount, err := image.DeploymentCount(testCtx, RawQuery{Query: paginatedQ.Query})
				assert.NoError(t, err)
				assert.Equal(t, int32(len(expectedDeployments)), actualDeploymentCount)

				// Test ImageCVECountBySeverity for each deployment resolver.
				expectedCVESevCount := compileExpectedCountBySeverity([]*storage.Image{expectedImages[imageID]}, tc.vulnFilter)
				actualCVECnt, err := image.ImageCVECountBySeverity(testCtx, RawQuery{Query: paginatedQ.Query})
				require.NoError(t, err)

				critical, err := actualCVECnt.Critical(testCtx)
				require.NoError(t, err)
				important, err := actualCVECnt.Important(testCtx)
				require.NoError(t, err)
				moderate, err := actualCVECnt.Moderate(testCtx)
				require.NoError(t, err)
				low, err := actualCVECnt.Low(testCtx)
				require.NoError(t, err)

				assert.Equal(t, int32(expectedCVESevCount.critical), critical.Total(testCtx))
				assert.Equal(t, int32(expectedCVESevCount.important), important.Total(testCtx))
				assert.Equal(t, int32(expectedCVESevCount.moderate), moderate.Total(testCtx))
				assert.Equal(t, int32(expectedCVESevCount.low), low.Total(testCtx))

				// Test BaseImage field
				actualBaseImage, err := image.BaseImage(testCtx)
				assert.NoError(t, err)

				expectedImage, exists := expectedImages[imageID]
				require.True(t, exists, "Expected image %s not found in expectedImages map", imageID)
				baseImageInfos := expectedImage.GetBaseImageInfo()
				if len(baseImageInfos) == 0 {
					assert.Nil(t, actualBaseImage)
				} else {
					require.NotNil(t, actualBaseImage)

					// Test imageSha (should be the digest from the first base image info)
					expectedSha := baseImageInfos[0].GetBaseImageDigest()
					assert.Equal(t, expectedSha, actualBaseImage.ImageSha(testCtx))

					// Test name array
					expectedNames := []string{
						baseImageInfos[1].GetBaseImageFullName(),
						baseImageInfos[0].GetBaseImageFullName(),
					}
					assert.Equal(t, expectedNames, actualBaseImage.Names(testCtx))

					// Test created timestamp
					actualCreated, err := actualBaseImage.Created(testCtx)
					assert.NoError(t, err)
					assert.NotNil(t, actualCreated)
					expectedTimestamp, err := protocompat.ConvertTimeToTimestampOrError(time.Unix(0, 3000))
					assert.NoError(t, err)
					expectedCreated, err := protocompat.ConvertTimestampToGraphqlTimeOrError(expectedTimestamp)
					assert.NoError(t, err)
					assert.Equal(t, expectedCreated, actualCreated)
				}

				// Test image -> deployments -> images
				imageDeployments, err := image.Deployments(testCtx, paginatedQ)
				assert.NoError(t, err)
				retrievedDeploymentIDs := make([]graphql.ID, 0, len(imageDeployments))
				expectedDeploymentIDs := make([]graphql.ID, 0, len(expectedDeployments))
				for _, d := range imageDeployments {
					retrievedDeploymentIDs = append(retrievedDeploymentIDs, d.Id(testCtx))
					deploymentImages, err := d.Images(testCtx, paginatedQ)
					assert.NoError(t, err)
					// Ensure deploymentImages only contains the caller/scoping image
					assert.Len(t, deploymentImages, 1)
					for _, img := range deploymentImages {
						assert.Equal(t, image.Id(testCtx), img.Id(testCtx))
					}
				}
				for _, d := range expectedDeployments {
					expectedDeploymentIDs = append(expectedDeploymentIDs, graphql.ID(d.GetId()))
				}
				assert.ElementsMatch(t, expectedDeploymentIDs, retrievedDeploymentIDs)

				// Test image -> image vulnerabilities -> images
				imageCVEs, err := image.ImageVulnerabilities(testCtx, paginatedQ)
				assert.NoError(t, err)
				for _, cve := range imageCVEs {
					cveImages, err := cve.Images(testCtx, paginatedQ)
					assert.NoError(t, err)
					assert.Len(t, cveImages, 1)
					for _, img := range cveImages {
						assert.Equal(t, image.Id(testCtx), img.Id(testCtx))
					}
				}
			}
		})
	}
}

const (
	// occurrenceSameImageCVE is fixable in comp-a1 and not fixable in comp-a2 of image A.
	occurrenceSameImageCVE = "cve-2030-1"
	// occurrenceCrossNamespaceCVE is fixable in image A, which only runs in namespace 1, and not
	// fixable in image B, which only runs in namespace 2.
	occurrenceCrossNamespaceCVE = "cve-2030-2"
	occurrenceAdvisory          = "RHSA-2030:0001"

	occurrenceImageA = "reg-occurrence/image-a:1.0"
	occurrenceImageB = "reg-occurrence/image-b:1.0"
)

// occurrenceTestImages adds CVE occurrences whose fixability differs within one image and across
// namespaces, which the shared image fixture does not have.
func occurrenceTestImages() []*storage.Image {
	image := func(remote string, components ...*storage.EmbeddedImageScanComponent) *storage.Image {
		return &storage.Image{
			Id: "sha256:occurrence-" + remote,
			Name: &storage.ImageName{
				Registry: "reg-occurrence",
				Remote:   remote,
				Tag:      "1.0",
				FullName: "reg-occurrence/" + remote + ":1.0",
			},
			Scan: &storage.ImageScan{Components: components},
		}
	}
	vuln := func(cve string, cvss float32, fixedBy string, severity storage.VulnerabilitySeverity) *storage.EmbeddedVulnerability {
		v := &storage.EmbeddedVulnerability{Cve: cve, Cvss: cvss, Severity: severity}
		if fixedBy != "" {
			v.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{FixedBy: fixedBy}
		}
		return v
	}
	important := storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	low := storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY

	// comp-a1's fixable occurrence has the highest CVSS, so it is image A's top vulnerability.
	fixableWithAdvisory := vuln(occurrenceSameImageCVE, 9.8, "1.1", important)
	fixableWithAdvisory.Advisory = &storage.Advisory{Name: occurrenceAdvisory, Link: "https://example.com/advisory"}
	return []*storage.Image{
		image("image-a",
			&storage.EmbeddedImageScanComponent{Name: "comp-a1", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{
				fixableWithAdvisory,
				vuln(occurrenceCrossNamespaceCVE, 5.0, "1.2", low),
			}},
			&storage.EmbeddedImageScanComponent{Name: "comp-a2", Version: "2.0", Vulns: []*storage.EmbeddedVulnerability{
				vuln(occurrenceSameImageCVE, 7.5, "", important),
			}},
		),
		image("image-b",
			&storage.EmbeddedImageScanComponent{Name: "comp-b1", Version: "1.0", Vulns: []*storage.EmbeddedVulnerability{
				vuln(occurrenceCrossNamespaceCVE, 5.0, "", low),
			}},
		),
	}
}

// occurrenceTestDeployments runs image A in cluster 1 / namespace 1 and image B in cluster 2 /
// namespace 2.
func occurrenceTestDeployments() []*storage.Deployment {
	images := occurrenceTestImages()
	deployment := func(id, name, clusterID, namespace, namespaceID string, image *storage.Image) *storage.Deployment {
		return &storage.Deployment{
			Id:          id,
			Name:        name,
			ClusterId:   clusterID,
			Namespace:   namespace,
			NamespaceId: namespaceID,
			Containers:  []*storage.Container{{Name: "container", Image: imageTypes.ToContainerImage(image)}},
		}
	}
	return []*storage.Deployment{
		deployment("deaaaaaa-bbbb-4011-0000-444444444444", "dep-occurrence-a", fixtureconsts.Cluster1, namespace1name, fixtureconsts.Namespace1, images[0]),
		deployment("deaaaaaa-bbbb-4011-0000-555555555555", "dep-occurrence-b", fixtureconsts.Cluster2, namespace2name, fixtureconsts.Namespace2, images[1]),
	}
}

// requestCtx returns a request context with a fresh loader cache, so data loaded with one access
// scope is not reused with another.
func (s *ImageResolversTestSuite) requestCtx(restricted bool) context.Context {
	ctx := sac.WithAllAccess(context.Background())
	if restricted {
		ctx = sacAllowOnlyCluster2Namespace2(context.Background())
	}
	return SetAuthorizerOverride(loaders.WithLoaderContext(ctx), allow.Anonymous())
}

// execGraphQL runs the query through the GraphQL engine and requires that it returns no errors.
func (s *ImageResolversTestSuite) execGraphQL(ctx context.Context, query string, variables map[string]interface{}, result interface{}) {
	s.T().Helper()
	response := s.schema.Exec(ctx, query, "", variables)
	s.Require().Empty(response.Errors)
	s.Require().NoError(json.Unmarshal(response.Data, result))
}

type occurrenceValues struct {
	IsFixable bool    `json:"isFixable"`
	EnvImpact float64 `json:"envImpact"`
}

func sortedFixability(values []occurrenceValues) []bool {
	fixable := make([]bool, 0, len(values))
	for _, value := range values {
		fixable = append(fixable, value.IsFixable)
	}
	sort.Slice(fixable, func(i, j int) bool { return !fixable[i] && fixable[j] })
	return fixable
}

func sortedEnvImpact(values []occurrenceValues) []float64 {
	envImpact := make([]float64, 0, len(values))
	for _, value := range values {
		envImpact = append(envImpact, value.EnvImpact)
	}
	slices.Sort(envImpact)
	return envImpact
}

// TestImageCVEsDistroTuples runs the imageCVEs distroTuples isFixable query. Each tuple represents a
// single CVE occurrence and must report that occurrence's fixability.
func (s *ImageResolversTestSuite) TestImageCVEsDistroTuples() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("requires the flattened image data model")
	}
	query := `query getImageCVEList($query: String, $pagination: Pagination) {
		imageCVEs(query: $query, pagination: $pagination) {
			cve
			distroTuples {
				isFixable
			}
		}
	}`
	variables := map[string]interface{}{"query": "", "pagination": map[string]interface{}{"limit": 20, "offset": 0}}
	for name, tc := range map[string]struct {
		restricted bool
		expected   map[string][]bool
	}{
		"unrestricted": {
			expected: map[string][]bool{
				"cve-2017-1":                {false},
				"cve-2017-2":                {false},
				"cve-2018-1":                {true, true, true},
				"cve-2019-1":                {false, false},
				"cve-2019-2":                {false, false},
				occurrenceSameImageCVE:      {false, true},
				occurrenceCrossNamespaceCVE: {false, true},
			},
		},
		// Namespace 2 runs image 2 and image B. Image 1 and image A, with the fixable
		// occurrence of the cross-namespace CVE, are hidden.
		"namespace 2": {
			restricted: true,
			expected: map[string][]bool{
				"cve-2017-1":                {false},
				"cve-2017-2":                {false},
				"cve-2018-1":                {true},
				"cve-2019-1":                {false},
				"cve-2019-2":                {false},
				occurrenceCrossNamespaceCVE: {false},
			},
		},
	} {
		s.Run(name, func() {
			var data struct {
				ImageCVEs []struct {
					CVE          string             `json:"cve"`
					DistroTuples []occurrenceValues `json:"distroTuples"`
				} `json:"imageCVEs"`
			}
			s.execGraphQL(s.requestCtx(tc.restricted), query, variables, &data)
			actual := make(map[string][]bool, len(data.ImageCVEs))
			for _, cve := range data.ImageCVEs {
				actual[cve.CVE] = sortedFixability(cve.DistroTuples)
			}
			s.Equal(tc.expected, actual)
		})
	}
}

// TestImageCVEDistroTuplesReturnOccurrenceFixability checks that each distro tuple of a CVE with
// mixed fixability reports its own occurrence's value.
func (s *ImageResolversTestSuite) TestImageCVEDistroTuplesReturnOccurrenceFixability() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("requires the flattened image data model")
	}
	ctx := s.requestCtx(false)
	core, err := s.resolver.ImageCVE(ctx, struct {
		Cve                *string
		SubfieldScopeQuery *string
	}{Cve: new(occurrenceSameImageCVE)})
	s.Require().NoError(err)
	s.Require().NotNil(core)

	tuples, err := core.DistroTuples(ctx)
	s.Require().NoError(err)
	s.Require().Len(tuples, 2)
	fixable := make([]bool, 0, len(tuples))
	for _, tuple := range tuples {
		occurrence := tuple.(*imageCVEV2Resolver).data
		isFixable, err := tuple.IsFixable(ctx, RawQuery{})
		s.Require().NoError(err)
		s.Equal(occurrence.GetFixedBy() != "", isFixable, "occurrence %s", occurrence.GetId())
		fixable = append(fixable, isFixable)
	}
	s.ElementsMatch([]bool{true, false}, fixable)
}

// TestImageCVEOccurrenceAndGroupedValues compares distro tuple values with grouped values, globally
// and within each image's scope. All 5 deployments are visible without restrictions; namespace 2
// sees deployment 3 (image 2) and the image B deployment.
func (s *ImageResolversTestSuite) TestImageCVEOccurrenceAndGroupedValues() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("requires the flattened image data model")
	}
	query := `query occurrenceAndGrouped($cve: String, $cveQuery: String) {
		imageCVE(cve: $cve) {
			distroTuples {
				isFixable
				envImpact
			}
		}
		imageVulnerabilities(query: $cveQuery) {
			isFixable
			envImpact
		}
		images {
			name {
				fullName
			}
			imageVulnerabilities(query: $cveQuery) {
				isFixable
				envImpact
			}
		}
	}`
	for name, tc := range map[string]struct {
		cve            string
		restricted     bool
		expectedTuples []occurrenceValues
		expectedGroup  occurrenceValues
		expectedImages map[string]occurrenceValues
	}{
		// Image 1 runs in deployments 1 and 2, image 2 in deployments 1 and 3.
		"shared fixture, unrestricted": {
			cve:            "cve-2018-1",
			expectedTuples: []occurrenceValues{{true, 2.0 / 5}, {true, 2.0 / 5}, {true, 2.0 / 5}},
			expectedGroup:  occurrenceValues{true, 3.0 / 5},
			expectedImages: map[string]occurrenceValues{
				"reg1/img1:tag1": {true, 2.0 / 5},
				"reg2/img2:tag2": {true, 2.0 / 5},
			},
		},
		"shared fixture, namespace 2": {
			cve:            "cve-2018-1",
			restricted:     true,
			expectedTuples: []occurrenceValues{{true, 1.0 / 2}},
			expectedGroup:  occurrenceValues{true, 1.0 / 2},
			expectedImages: map[string]occurrenceValues{"reg2/img2:tag2": {true, 1.0 / 2}},
		},
		"mixed fixability in one image": {
			cve:            occurrenceSameImageCVE,
			expectedTuples: []occurrenceValues{{false, 1.0 / 5}, {true, 1.0 / 5}},
			expectedGroup:  occurrenceValues{true, 1.0 / 5},
			expectedImages: map[string]occurrenceValues{occurrenceImageA: {true, 1.0 / 5}},
		},
		"mixed fixability across namespaces, unrestricted": {
			cve:            occurrenceCrossNamespaceCVE,
			expectedTuples: []occurrenceValues{{false, 1.0 / 5}, {true, 1.0 / 5}},
			expectedGroup:  occurrenceValues{true, 2.0 / 5},
			expectedImages: map[string]occurrenceValues{
				occurrenceImageA: {true, 1.0 / 5},
				occurrenceImageB: {false, 1.0 / 5},
			},
		},
		// The fixable occurrence in namespace 1 must not affect any visible value.
		"mixed fixability across namespaces, namespace 2": {
			cve:            occurrenceCrossNamespaceCVE,
			restricted:     true,
			expectedTuples: []occurrenceValues{{false, 1.0 / 2}},
			expectedGroup:  occurrenceValues{false, 1.0 / 2},
			expectedImages: map[string]occurrenceValues{occurrenceImageB: {false, 1.0 / 2}},
		},
	} {
		s.Run(name, func() {
			var data struct {
				ImageCVE struct {
					DistroTuples []occurrenceValues `json:"distroTuples"`
				} `json:"imageCVE"`
				ImageVulnerabilities []occurrenceValues `json:"imageVulnerabilities"`
				Images               []struct {
					Name struct {
						FullName string `json:"fullName"`
					} `json:"name"`
					ImageVulnerabilities []occurrenceValues `json:"imageVulnerabilities"`
				} `json:"images"`
			}
			s.execGraphQL(s.requestCtx(tc.restricted), query, map[string]interface{}{
				"cve":      tc.cve,
				"cveQuery": "CVE:" + tc.cve,
			}, &data)

			s.Equal(sortedFixability(tc.expectedTuples), sortedFixability(data.ImageCVE.DistroTuples))
			s.InDeltaSlice(sortedEnvImpact(tc.expectedTuples), sortedEnvImpact(data.ImageCVE.DistroTuples), 1e-9)

			s.Require().Len(data.ImageVulnerabilities, 1)
			s.Equal(tc.expectedGroup.IsFixable, data.ImageVulnerabilities[0].IsFixable)
			s.InDelta(tc.expectedGroup.EnvImpact, data.ImageVulnerabilities[0].EnvImpact, 1e-9)

			actualImages := make(map[string]occurrenceValues)
			for _, image := range data.Images {
				if len(image.ImageVulnerabilities) == 0 {
					continue
				}
				s.Require().Len(image.ImageVulnerabilities, 1, image.Name.FullName)
				actualImages[image.Name.FullName] = image.ImageVulnerabilities[0]
			}
			s.Require().Len(actualImages, len(tc.expectedImages))
			for image, expected := range tc.expectedImages {
				s.Equal(expected.IsFixable, actualImages[image].IsFixable, image)
				s.InDelta(expected.EnvImpact, actualImages[image].EnvImpact, 1e-9, image)
			}
		})
	}
}

// TestImageTopVulnerabilities checks top vulnerabilities, which represent a single CVE occurrence,
// in image and component scope.
func (s *ImageResolversTestSuite) TestImageTopVulnerabilities() {
	if !features.FlattenImageData.Enabled() {
		s.T().Skip("requires the flattened image data model")
	}
	type topVulnerability struct {
		CVE            string `json:"cve"`
		FixedByVersion string `json:"fixedByVersion"`
		Advisory       *struct {
			Name string `json:"name"`
		} `json:"advisory"`
		IsFixable bool `json:"isFixable"`
	}
	var data struct {
		Images []struct {
			Name struct {
				FullName string `json:"fullName"`
			} `json:"name"`
			TopImageVulnerability topVulnerability `json:"topImageVulnerability"`
			ImageComponents       []struct {
				Name                  string           `json:"name"`
				TopImageVulnerability topVulnerability `json:"topImageVulnerability"`
			} `json:"imageComponents"`
		} `json:"images"`
	}
	s.execGraphQL(s.requestCtx(false), `query topVulnerabilities {
		images {
			name {
				fullName
			}
			topImageVulnerability {
				cve
				isFixable
			}
			imageComponents {
				name
				topImageVulnerability {
					cve
					fixedByVersion
					advisory {
						name
					}
					isFixable
				}
			}
		}
	}`, nil, &data)

	type expectedTop struct {
		cve            string
		fixedByVersion string
		advisory       string
		isFixable      bool
	}
	toExpectedTop := func(top topVulnerability) expectedTop {
		actual := expectedTop{cve: top.CVE, fixedByVersion: top.FixedByVersion, isFixable: top.IsFixable}
		if top.Advisory != nil {
			actual.advisory = top.Advisory.Name
		}
		return actual
	}
	actualImages := make(map[string]expectedTop, len(data.Images))
	actualComponents := make(map[string]expectedTop)
	for _, image := range data.Images {
		actualImages[image.Name.FullName] = toExpectedTop(image.TopImageVulnerability)
		if image.Name.FullName == occurrenceImageA || image.Name.FullName == occurrenceImageB {
			for _, component := range image.ImageComponents {
				actualComponents[component.Name] = toExpectedTop(component.TopImageVulnerability)
			}
		}
	}
	s.Equal(map[string]expectedTop{
		"reg1/img1:tag1": {cve: "cve-2019-1"},
		"reg2/img2:tag2": {cve: "cve-2019-1"},
		occurrenceImageA: {cve: occurrenceSameImageCVE, isFixable: true},
		occurrenceImageB: {cve: occurrenceCrossNamespaceCVE},
	}, actualImages)
	s.Equal(map[string]expectedTop{
		"comp-a1": {cve: occurrenceSameImageCVE, fixedByVersion: "1.1", advisory: occurrenceAdvisory, isFixable: true},
		"comp-a2": {cve: occurrenceSameImageCVE},
		"comp-b1": {cve: occurrenceCrossNamespaceCVE},
	}, actualComponents)
}

// TODO(ROX-30117): Remove conditional when FlattenImageData feature flag is removed.
func getImageIDForTest(image *storage.Image) string {
	if features.FlattenImageData.Enabled() {
		return imageUtils.ConvertToV2(image).GetId()
	}
	return image.GetId()
}

func compileExpectedForImageGraphQL(deployments []*storage.Deployment, images []*storage.Image,
	deploymentFilter func(d *storage.Deployment) bool,
	imageFilter func(d *storage.Image) bool) (map[string]*storage.Image, map[string][]*storage.Deployment) {
	imageToDeploymentsMap := make(map[string][]*storage.Deployment)
	for _, deployment := range deployments {
		for _, container := range deployment.GetContainers() {
			imgName := container.GetImage().GetName().GetFullName()
			imageToDeploymentsMap[imgName] = append(imageToDeploymentsMap[imgName], deployment)
		}
	}

	matchedImages := make(map[string]*storage.Image)
	matchedDeploymentsPerImage := make(map[string][]*storage.Deployment)
	for _, image := range images {
		if !imageFilter(image) {
			continue
		}

		imageID := getImageIDForTest(image)
		var deploymentFilterPassed bool
		for _, deployment := range imageToDeploymentsMap[image.GetName().GetFullName()] {
			if deployment == nil {
				continue
			}
			if !deploymentFilter(deployment) {
				continue
			}
			deploymentFilterPassed = true
			matchedDeploymentsPerImage[imageID] = append(matchedDeploymentsPerImage[imageID], deployment)
		}

		if deploymentFilterPassed {
			matchedImages[imageID] = image
		}
	}

	return matchedImages, matchedDeploymentsPerImage
}
