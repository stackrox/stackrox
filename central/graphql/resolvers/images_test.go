//go:build sql_integration

package resolvers

import (
	"context"
	"strings"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/central/views/imagecve"
	imagesView "github.com/stackrox/rox/central/views/images"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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

	testDeployments []*storage.Deployment
	testImages      []*storage.Image
}

func (s *ImageResolversTestSuite) SetupSuite() {
	s.ctx = loaders.WithLoaderContext(sac.WithAllAccess(context.Background()))
	mockCtrl := gomock.NewController(s.T())
	s.testDB = SetupTestPostgresConn(s.T())
	imgDataStore := CreateTestImageDatastore(s.T(), s.testDB, mockCtrl)
	resolver, _ := SetupTestResolver(s.T(),
		CreateTestDeploymentDatastore(s.T(), s.testDB, mockCtrl, imgDataStore),
		imagesView.NewImageView(s.testDB.DB),
		imgDataStore,
		CreateTestImageComponentDatastore(s.T(), s.testDB, mockCtrl),
		CreateTestImageCVEDatastore(s.T(), s.testDB),
		imagecve.NewCVEView(s.testDB.DB),
	)
	s.resolver = resolver

	// Add Test Data.
	s.testDeployments = testDeployments()
	for _, deployment := range s.testDeployments {
		s.NoError(s.resolver.DeploymentDataStore.UpsertDeployment(s.ctx, deployment))
	}
	s.testImages = testImages()
	for _, image := range testImages() {
		s.NoError(s.resolver.ImageDataStore.UpsertImage(s.ctx, image))
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
			q:    PaginatedQuery{Query: pointers.String("Namespace:namespace1name")},
			deploymentFiler: func(d *storage.Deployment) bool {
				return strings.HasPrefix(d.GetNamespace(), "namespace1name")
			},
			imageFilter: func(_ *storage.Image) bool { return true },
			vulnFilter:  func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc: "filter by deployment",
			ctx:  ctx,
			q:    PaginatedQuery{Query: pointers.String("Deployment:" + dep1name)},
			deploymentFiler: func(d *storage.Deployment) bool {
				return d.GetName() == dep1name
			},
			imageFilter: func(_ *storage.Image) bool { return true },
			vulnFilter:  func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc:            "filter by image",
			ctx:             ctx,
			q:               PaginatedQuery{Query: pointers.String("Image:reg1/img1")},
			deploymentFiler: func(d *storage.Deployment) bool { return true },
			imageFilter: func(img *storage.Image) bool {
				return strings.HasPrefix(img.GetName().GetFullName(), "reg1/img1")
			},
			vulnFilter: func(_ *storage.EmbeddedVulnerability) bool { return true },
		},
		{
			desc:            "filter by cve",
			ctx:             ctx,
			q:               PaginatedQuery{Query: pointers.String("CVE:cve-2019-2")},
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
			q:    PaginatedQuery{Query: pointers.String("Deployment:dep2name+CVE:cve-2019-2")},
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
			q:               PaginatedQuery{Query: pointers.String("Severity:CRITICAL_VULNERABILITY_SEVERITY")},
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
			q:               PaginatedQuery{Query: pointers.String("Severity:UNSET_VULNERABILITY_SEVERITY+Fixable:true")},
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
			for _, dep := range expectedImages {
				expectedIDs = append(expectedIDs, dep.GetId())
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
					expectedDeploymentIDs = append(expectedDeploymentIDs, graphql.ID(d.Id))
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

		var deploymentFilterPassed bool
		for _, deployment := range imageToDeploymentsMap[image.GetName().GetFullName()] {
			if deployment == nil {
				continue
			}
			if !deploymentFilter(deployment) {
				continue
			}
			deploymentFilterPassed = true
			matchedDeploymentsPerImage[image.GetId()] = append(matchedDeploymentsPerImage[image.GetId()], deployment)
		}

		if deploymentFilterPassed {
			matchedImages[image.GetId()] = image
		}
	}

	return matchedImages, matchedDeploymentsPerImage
}
