//go:build sql_integration

package imagecveflat

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	imageCVEV2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageComponentV2DS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	imageSamples "github.com/stackrox/rox/pkg/fixtures/image"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type testCase struct {
	desc           string
	ctx            context.Context
	q              *v1.Query
	matchFilter    *filterImpl
	less           lessFunc
	readOptions    views.ReadOptions
	expectedErr    string
	skipCountTests bool
	testOrder      bool
}

type lessFunc func(records []*imageCVEFlatResponse) func(i, j int) bool

// testImage is satisfied by both *storage.Image and *storage.ImageV2.
type testImage interface {
	GetId() string
	GetName() *storage.ImageName
	GetScan() *storage.ImageScan
}

type filterImpl struct {
	matchImage func(image testImage) bool
	matchVuln  func(cve *storage.ImageCVEV2) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ testImage) bool {
			return true
		},
		matchVuln: func(_ *storage.ImageCVEV2) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ testImage) bool {
			return false
		},
		matchVuln: func(_ *storage.ImageCVEV2) bool {
			return false
		},
	}
}

func (f *filterImpl) withImageFilter(fn func(image testImage) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withVulnFilter(fn func(vuln *storage.ImageCVEV2) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestImageCVEVFlatiew(t *testing.T) {
	suite.Run(t, new(ImageCVEFlatViewTestSuite))
}

type ImageCVEFlatViewTestSuite struct {
	suite.Suite

	testDB   *pgtest.TestPostgres
	cveView  CveFlatView
	suiteCtx context.Context

	testImages              []testImage
	testImagesToDeployments map[string][]*storage.Deployment

	componentDatastore imageComponentV2DS.DataStore
	cveDatastore       imageCVEV2DS.DataStore
}

func (s *ImageCVEFlatViewTestSuite) SetupSuite() {
	s.suiteCtx = sac.WithAllAccess(context.Background())
	ctx := s.suiteCtx
	s.testDB = pgtest.ForT(s.T())

	// Initialize the datastores.
	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)
	s.componentDatastore = imageComponentV2DS.GetTestPostgresDataStore(s.T(), s.testDB)
	s.cveDatastore = imageCVEV2DS.GetTestPostgresDataStore(s.T(), s.testDB)

	// setCVSSMetrics sets NVD CVSS metrics on all vulns in an image scan.
	setCVSSMetrics := func(scan *storage.ImageScan) {
		for _, component := range scan.GetComponents() {
			for _, vuln := range component.GetVulns() {
				vuln.CvssMetrics = []*storage.CVSSScore{{
					Source: storage.Source_SOURCE_NVD,
					CvssScore: &storage.CVSSScore_Cvssv3{
						Cvssv3: &storage.CVSSV3{Score: 10},
					},
				}}
				vuln.NvdCvss = 10
			}
		}
	}

	// Upsert images using the appropriate datastore based on feature flag.
	var deployments []*storage.Deployment
	if features.FlattenImageData.Enabled() {
		imageV2Store := imageV2DS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
		imagesV2, err := imageSamples.GetTestImagesV2(s.T())
		s.Require().NoError(err)
		for _, imgV2 := range imagesV2 {
			setCVSSMetrics(imgV2.GetScan())
			s.Require().NoError(imageV2Store.UpsertImage(ctx, imgV2))
		}
		// Verify stored V2 images and use them for expected results.
		for idx, imgV2 := range imagesV2 {
			actual, found, err := imageV2Store.GetImage(ctx, imgV2.GetId())
			s.Require().NoError(err)
			s.Require().True(found)
			imagesV2[idx] = actual
		}
		s.testImages = make([]testImage, len(imagesV2))
		for i, img := range imagesV2 {
			s.testImages[i] = img
		}
		deployments = []*storage.Deployment{
			fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, testconsts.NamespaceA, imagesV2[1]),
			fixtures.GetDeploymentWithImageV2(testconsts.Cluster2, testconsts.NamespaceB, imagesV2[1]),
			fixtures.GetDeploymentWithImageV2(testconsts.Cluster2, testconsts.NamespaceB, imagesV2[2]),
		}
	} else {
		imageStore := imageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
		images, err := imageSamples.GetTestImages(s.T())
		s.Require().NoError(err)
		for _, image := range images {
			setCVSSMetrics(image.GetScan())
			s.Require().NoError(imageStore.UpsertImage(ctx, image))
		}
		// Ensure that the image is stored and constructed as expected.
		for idx, image := range images {
			actual, found, err := imageStore.GetImage(ctx, image.GetId())
			s.Require().NoError(err)
			s.Require().True(found)

			cloned := actual.CloneVT()
			standardizeImages(image, cloned)

			images[idx] = actual
		}
		s.testImages = make([]testImage, len(images))
		for i, img := range images {
			s.testImages[i] = img
		}
		deployments = []*storage.Deployment{
			fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, images[1]),
			fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[1]),
			fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[2]),
		}
	}

	s.cveView = NewCVEFlatView(s.testDB.DB)

	s.Require().Len(s.testImages, 5)
	for _, d := range deployments {
		s.Require().NoError(deploymentStore.UpsertDeployment(ctx, d))
	}

	s.testImagesToDeployments = make(map[string][]*storage.Deployment)
	s.testImagesToDeployments[s.testImages[1].GetId()] = []*storage.Deployment{deployments[0], deployments[1]}
	s.testImagesToDeployments[s.testImages[2].GetId()] = []*storage.Deployment{deployments[2]}
}

// imageScopeCategory returns the search category for image scoping.
func (s *ImageCVEFlatViewTestSuite) imageScopeCategory() v1.SearchCategory {
	if features.FlattenImageData.Enabled() {
		return v1.SearchCategory_IMAGES_V2
	}
	return v1.SearchCategory_IMAGES
}

// imageSearchField returns the search field label for image ID.
func imageSearchField() search.FieldLabel {
	if features.FlattenImageData.Enabled() {
		return search.ImageID
	}
	return search.ImageSHA
}

func (s *ImageCVEFlatViewTestSuite) findImageByName(fullName string) testImage {
	for _, img := range s.testImages {
		if img.GetName().GetFullName() == fullName {
			return img
		}
	}
	return nil
}

func (s *ImageCVEFlatViewTestSuite) TestGetImageCVEFlat() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.Get(sac.WithAllAccess(tc.ctx), tc.q, tc.readOptions)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpected(s.testImages, tc.matchFilter, tc.readOptions, tc.less)
			assert.Equal(t, len(expected), len(actual))
			assertResponsesAreEqual(t, expected, actual, tc.testOrder)

			if tc.readOptions.SkipGetAffectedImages || tc.readOptions.SkipGetImagesBySeverity {
				return
			}
		})
	}
}

func (s *ImageCVEFlatViewTestSuite) TestGetImageCVEFlatSAC() {
	for _, tc := range s.testCases() {
		for key, sacTC := range s.sacTestCases() {
			s.T().Run(fmt.Sprintf("Image %s %s", key, tc.desc), func(t *testing.T) {
				testCtxs := testutils.GetNamespaceScopedTestContexts(tc.ctx, s.T(), resources.Image)
				ctx := testCtxs[key]

				actual, err := s.cveView.Get(ctx, tc.q, tc.readOptions)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Wrap image filter with sac filter.
				matchFilter := *tc.matchFilter
				baseImageMatchFilter := matchFilter.matchImage
				matchFilter.withImageFilter(func(image testImage) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expected := s.compileExpected(s.testImages, &matchFilter, tc.readOptions, tc.less)
				assert.Equal(t, len(expected), len(actual))
				assertResponsesAreEqual(t, expected, actual, false)
			})
		}
	}

	// Testing one query against deployment access is sufficient.
	tc := s.testCases()[0]
	s.T().Run(fmt.Sprintf("Deployment read access %s", tc.desc), func(t *testing.T) {
		ctx := sac.WithGlobalAccessScopeChecker(tc.ctx,
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
				sac.ResourceScopeKeys(resources.Deployment)))

		actual, err := s.cveView.Get(ctx, tc.q, tc.readOptions)
		if tc.expectedErr != "" {
			s.ErrorContains(err, tc.expectedErr)
			return
		}
		assert.NoError(t, err)
		assertResponsesAreEqual(t, []CveFlat{}, actual, false)
	})
}

func (s *ImageCVEFlatViewTestSuite) TestGetImageCVEFlatWithPagination() {
	for _, paginationTestCase := range s.paginationTestCases() {
		baseTestCases := s.testCases()
		for idx := range baseTestCases {
			tc := &baseTestCases[idx]
			if !tc.readOptions.IsDefault() {
				continue
			}
			applyPaginationProps(tc, paginationTestCase)

			s.T().Run(tc.desc, func(t *testing.T) {
				actual, err := s.cveView.Get(sac.WithAllAccess(tc.ctx), tc.q, tc.readOptions)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				expected := s.compileExpected(s.testImages, tc.matchFilter, tc.readOptions, tc.less)

				assert.Equal(t, len(expected), len(actual))
				assertResponsesAreEqual(t, expected, actual, true)

				if tc.readOptions.SkipGetAffectedImages || tc.readOptions.SkipGetImagesBySeverity {
					return
				}
			})
		}
	}
}

func (s *ImageCVEFlatViewTestSuite) TestCountImageCVEFlat() {
	for _, tc := range s.testCases() {
		if tc.skipCountTests {
			continue
		}

		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.Count(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpected(s.testImages, tc.matchFilter, tc.readOptions, nil)
			assert.Equal(t, len(expected), actual)
		})
	}
}

func (s *ImageCVEFlatViewTestSuite) TestCountImageCVEFlatSAC() {
	for _, tc := range s.testCases() {
		for key, sacTC := range s.sacTestCases() {
			if tc.skipCountTests {
				continue
			}

			s.T().Run(fmt.Sprintf("Image %s %s", key, tc.desc), func(t *testing.T) {
				testCtxs := testutils.GetNamespaceScopedTestContexts(tc.ctx, s.T(), resources.Image)
				ctx := testCtxs[key]

				actual, err := s.cveView.Count(ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Wrap image filter with sac filter.
				matchFilter := *tc.matchFilter
				baseImageMatchFilter := matchFilter.matchImage
				matchFilter.withImageFilter(func(image testImage) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expected := s.compileExpected(s.testImages, &matchFilter, tc.readOptions, tc.less)
				assert.Equal(t, len(expected), actual)
			})
		}
	}
}

func (s *ImageCVEFlatViewTestSuite) testCases() []testCase {
	wordpressDebian := s.findImageByName("quay.io/appcontainers/wordpress:debian")
	s.Require().NotNil(wordpressDebian)
	wordpressLatest := s.findImageByName("quay.io/appcontainers/wordpress:latest")
	s.Require().NotNil(wordpressLatest)

	return []testCase{
		{
			desc:        "search all",
			ctx:         context.Background(),
			q:           search.EmptyQuery(),
			matchFilter: matchAllFilter(),
		},
		{
			desc: "search one cve",
			ctx:  context.Background(),
			q:    search.NewQueryBuilder().AddExactMatches(search.CVE, "CVE-2022-1552").ProtoQuery(),
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
				return vuln.GetCveBaseInfo().GetCve() == "CVE-2022-1552"
			}),
		},
		{
			desc: "search one image",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:latest").ProtoQuery(),
			matchFilter: matchAllFilter().withImageFilter(func(image testImage) bool {
				return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:latest"
			}),
		},
		{
			desc: "search one cve + one image",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2022-1552").
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:debian").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image testImage) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
				}).
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2022-1552"
				}),
		},
		{
			desc: "search critical severity",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc: "search fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddBools(search.Fixable, true).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetFixedBy() != ""
				}),
		},
		{
			desc: "search one cve + fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2015-8704").
				AddBools(search.Fixable, true).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2015-8704" && vuln.GetFixedBy() != ""
				}),
		},
		{
			desc: "search one cve + not fixable",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2015-8704").
				AddBools(search.Fixable, false).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2015-8704" && vuln.GetFixedBy() == ""
				}),
		},
		{
			desc: "search multiple severities",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity,
					storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String(),
					storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY.String(),
				).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY ||
						vuln.GetSeverity() == storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc: "search critical severity + one image",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:debian").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image testImage) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
				}).
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc:        "no match",
			ctx:         context.Background(),
			q:           search.NewQueryBuilder().AddExactMatches(search.Component, "").ProtoQuery(),
			matchFilter: matchNoneFilter(),
		},
		{
			desc: "with select",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.CVE)).
				AddExactMatches(search.OperatingSystem, "").ProtoQuery(),
			expectedErr: "Unexpected select clause in query",
		},
		{
			desc: "with group by",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.OperatingSystem, "").
				AddGroupBy(search.CVE).ProtoQuery(),
			expectedErr: "Unexpected group by clause in query",
		},
		{
			desc:        "search all; skip top cvss; skip images by severity",
			ctx:         context.Background(),
			q:           search.NewQueryBuilder().ProtoQuery(),
			matchFilter: matchAllFilter(),
			readOptions: views.ReadOptions{
				SkipGetImagesBySeverity: true,
				SkipGetTopCVSS:          true,
			},
		},
		{
			desc: "search one cve w/ image scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				IDs:   []string{wordpressDebian.GetId()},
				Level: s.imageScopeCategory(),
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2022-1552").
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:debian").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image testImage) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
				}).
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2022-1552"
				}),
		},
		{
			desc: "search critical severity w/ cve & image scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				IDs:   []string{wordpressDebian.GetId()},
				Level: s.imageScopeCategory(),
				Parent: &scoped.Scope{
					IDs: []string{getTestCVEID(getTestCVE(),
						getTestComponentID(&storage.EmbeddedImageScanComponent{
							Name:         "postgresql-libs",
							Version:      "8.4.20-6.el6",
							Source:       storage.SourceType_OS,
							Location:     "",
							Architecture: "",
						}, wordpressLatest.GetId(), 0), 0)},
					Level: v1.SearchCategory_IMAGE_VULNERABILITIES_V2,
				},
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image testImage) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian" &&
						image.GetScan().GetOperatingSystem() == "debian:8"
				}).
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetCveBaseInfo().GetCve() == "CVE-2022-1552" &&
						vuln.GetSeverity() == storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
				}),
		},
		{
			desc: "search observed CVEs from inactive images and active images in non-platform deployments",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
				AddStrings(search.PlatformComponent, "false", "-").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image testImage) bool {
					deps, ok := s.testImagesToDeployments[image.GetId()]
					if !ok {
						// include inactive image
						return true
					}
					for _, d := range deps {
						if !d.GetPlatformComponent() {
							return true
						}
					}
					return false
				}).
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetState() == storage.VulnerabilityState_OBSERVED
				}),
		},
		{
			desc: "search observed CVEs from inactive images and active images in platform deployments",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.VulnerabilityState, storage.VulnerabilityState_OBSERVED.String()).
				AddStrings(search.PlatformComponent, "true", "-").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image testImage) bool {
					deps, ok := s.testImagesToDeployments[image.GetId()]
					if !ok {
						// include inactive image
						return true
					}
					for _, d := range deps {
						if d.GetPlatformComponent() {
							return true
						}
					}
					return false
				}).
				withVulnFilter(func(vuln *storage.ImageCVEV2) bool {
					return vuln.GetState() == storage.VulnerabilityState_OBSERVED
				}),
		},
	}
}

func (s *ImageCVEFlatViewTestSuite) paginationTestCases() []testCase {
	return []testCase{
		{
			desc: "w/ affected image sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(imageSearchField()).AggregateBy(aggregatefunc.Count, true).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			less: func(records []*imageCVEFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					if records[i].GetAffectedImageCount() == records[j].GetAffectedImageCount() {
						return records[i].CVE < records[j].CVE
					}
					return records[i].GetAffectedImageCount() > records[j].GetAffectedImageCount()
				}
			},
		},
		{
			desc: "w/ top cvss sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.CVSS).AggregateBy(aggregatefunc.Max, false).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			less: func(records []*imageCVEFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					if records[i].GetTopCVSS() == records[j].GetTopCVSS() {
						return records[i].CVE < records[j].CVE
					}
					return records[i].GetTopCVSS() > records[j].GetTopCVSS()
				}
			},
		},
		{
			desc: "w/ first discovered sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.CVECreatedTime).AggregateBy(aggregatefunc.Min, false),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			less: func(records []*imageCVEFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					recordI, recordJ := records[i], records[j]
					if recordJ == nil {
						recordJ = &imageCVEFlatResponse{}
					}
					if recordI.FirstDiscoveredInSystem.Equal(*recordJ.FirstDiscoveredInSystem) {
						return records[i].CVE < records[j].CVE
					}
					return recordI.FirstDiscoveredInSystem.Before(*recordJ.FirstDiscoveredInSystem)
				}
			},
		},
		{
			desc: "w/ epss probability sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.EPSSProbablity).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			less: func(records []*imageCVEFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					recordI, recordJ := records[i], records[j]
					if recordJ == nil {
						recordJ = &imageCVEFlatResponse{}
					}
					if recordI.GetEPSSProbability() == recordJ.GetEPSSProbability() {
						return records[i].CVE < records[j].CVE
					}
					return recordI.GetEPSSProbability() > recordJ.GetEPSSProbability()
				}
			},
		},
	}
}

func (s *ImageCVEFlatViewTestSuite) sacTestCases() map[string]map[string]bool {
	s.Require().Len(s.testImages, 5)

	img1 := s.testImages[0]
	img2 := s.testImages[1]
	img3 := s.testImages[2]
	img4 := s.testImages[3]
	img5 := s.testImages[4]

	// The map structure is the mapping ScopeKey -> ImageID -> Visible
	return map[string]map[string]bool{
		testutils.UnrestrictedReadCtx: {
			img1.GetId(): true,
			img2.GetId(): true,
			img3.GetId(): true,
			img4.GetId(): true,
			img5.GetId(): true,
		},
		testutils.UnrestrictedReadWriteCtx: {
			img1.GetId(): true,
			img2.GetId(): true,
			img3.GetId(): true,
			img4.GetId(): true,
			img5.GetId(): true,
		},
		testutils.Cluster1ReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespaceAReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespaceBReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespacesABReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespacesBCReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster2ReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true,
			img3.GetId(): true,
		},
		testutils.Cluster2NamespaceAReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster2NamespaceBReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true,
			img3.GetId(): true,
		},
		testutils.Cluster2NamespacesACReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster2NamespacesBCReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true,
			img3.GetId(): true,
		},
		testutils.Cluster3ReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster3NamespaceAReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster3NamespaceBReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
	}
}

func applyPaginationProps(baseTc *testCase, paginationTc testCase) {
	baseTc.desc = fmt.Sprintf("%s %s", baseTc.desc, paginationTc.desc)
	baseTc.q.Pagination = paginationTc.q.GetPagination()
	baseTc.less = paginationTc.less
}

func (s *ImageCVEFlatViewTestSuite) compileExpected(images []testImage, filter *filterImpl, options views.ReadOptions, less lessFunc) []CveFlat {
	cveMap := make(map[string]*imageCVEFlatResponse)

	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		var seenForImage set.Set[string]
		components, err := s.componentDatastore.SearchRawImageComponents(s.suiteCtx, search.NewQueryBuilder().AddExactMatches(imageSearchField(), image.GetId()).ProtoQuery())
		s.Require().NoError(err)
		// Instead of rebuilding these from what we return in the image, grab them from the component and cve store
		for _, component := range components {
			dbVulns, err := s.cveDatastore.SearchRawImageCVEs(s.suiteCtx, search.NewQueryBuilder().AddExactMatches(search.ComponentID, component.GetId()).ProtoQuery())
			s.Require().NoError(err)
			for _, vuln := range dbVulns {
				if !filter.matchVuln(vuln) {
					continue
				}

				vulnTime, _ := protocompat.ConvertTimestampToTimeOrError(vuln.GetCveBaseInfo().GetCreatedAt())
				vulnTime = vulnTime.Round(time.Microsecond)
				vulnPublishDate, _ := protocompat.ConvertTimestampToTimeOrError(vuln.GetCveBaseInfo().GetPublishedOn())
				vulnPublishDate = vulnPublishDate.Round(time.Microsecond)
				vulnImageOccurrence, _ := protocompat.ConvertTimestampToTimeOrError(vuln.GetFirstImageOccurrence())
				vulnImageOccurrence = vulnImageOccurrence.Round(time.Microsecond)
				val := cveMap[vuln.GetCveBaseInfo().GetCve()]

				var impactScore float32
				if vuln.GetCveBaseInfo().GetCvssV3() != nil {
					impactScore = vuln.GetCveBaseInfo().GetCvssV3().GetImpactScore()
				} else if vuln.GetCveBaseInfo().GetCvssV2() != nil {
					impactScore = vuln.GetCveBaseInfo().GetCvssV2().GetImpactScore()
				}

				if val == nil {
					val = &imageCVEFlatResponse{
						CVE:                     vuln.GetCveBaseInfo().GetCve(),
						TopCVSS:                 pointers.Float32(vuln.GetCvss()),
						FirstDiscoveredInSystem: &vulnTime,
						Published:               &vulnPublishDate,
						Severity:                vuln.GetSeverity().Enum(),
						FirstImageOccurrence:    &vulnImageOccurrence,
						State:                   pointers.Pointer(vuln.GetState()),
						ImpactScore:             pointers.Float32(impactScore),
						EpssProbability:         pointers.Float32(vuln.GetCveBaseInfo().GetEpss().GetEpssProbability()),
					}
					for _, metric := range vuln.GetCveBaseInfo().GetCvssMetrics() {
						if metric.GetSource() == storage.Source_SOURCE_NVD {
							if metric.GetCvssv2() != nil {
								val.TopNVDCVSS = pointers.Float32(metric.GetCvssv2().GetScore())
							} else {
								val.TopNVDCVSS = pointers.Float32(metric.GetCvssv3().GetScore())
							}
						}
					}
					cveMap[val.CVE] = val
				}

				id := vuln.GetId()
				var found bool
				for _, seenID := range val.GetCVEIDs() {
					if seenID == id {
						found = true
						break
					}
				}

				if !found {
					val.CVEIDs = append(val.CVEIDs, id)
				}

				val.TopCVSS = pointers.Float32(max(val.GetTopCVSS(), vuln.GetCvss()))
				val.ImpactScore = pointers.Float32(max(*val.ImpactScore, impactScore))
				val.EpssProbability = pointers.Float32(max(*val.EpssProbability, vuln.GetCveBaseInfo().GetEpss().GetEpssProbability()))
				if vuln.GetSeverity().Number() > val.GetSeverity().Number() {
					val.Severity = pointers.Pointer(vuln.GetSeverity())
				}

				if val.GetFirstDiscoveredInSystem().After(vulnTime) {
					val.FirstDiscoveredInSystem = &vulnTime
				}
				if val.GetPublishDate().After(vulnPublishDate) {
					val.Published = &vulnPublishDate
				}
				if val.GetFirstImageOccurrence().After(vulnImageOccurrence) {
					val.FirstImageOccurrence = &vulnImageOccurrence
				}

				if !seenForImage.Add(val.CVE) {
					continue
				}
				if features.FlattenImageData.Enabled() {
					val.AffectedImageCountV2++
				} else {
					val.AffectedImageCount++
				}
			}
		}
	}

	expected := make([]*imageCVEFlatResponse, 0, len(cveMap))

	for _, entry := range cveMap {
		expected = append(expected, entry)
	}
	if options.SkipGetTopCVSS {
		for _, entry := range expected {
			entry.TopCVSS = nil
		}
	}
	if options.SkipGetAffectedImages {
		for _, entry := range cveMap {
			entry.AffectedImageCount = 0
			entry.AffectedImageCountV2 = 0
		}
	}
	if options.SkipGetFirstDiscoveredInSystem {
		for _, entry := range cveMap {
			entry.FirstDiscoveredInSystem = nil
		}
	}
	if less != nil {
		sort.SliceStable(expected, less(expected))
	}

	ret := make([]CveFlat, 0, len(cveMap))
	for _, entry := range expected {
		ret = append(ret, entry)
	}
	return ret
}

func standardizeImages(images ...*storage.Image) {
	for _, image := range images {
		if image.GetMetadata().GetV1() != nil && len(image.GetMetadata().GetV1().GetLabels()) == 0 {
			image.Metadata.V1.Labels = nil
		}

		components := image.GetScan().GetComponents()
		for _, component := range components {
			component.Priority = 0
			if len(component.GetVulns()) == 0 {
				component.Vulns = nil
			}

			vulns := component.GetVulns()
			for _, vuln := range vulns {
				vuln.FirstImageOccurrence = nil
				vuln.FirstSystemOccurrence = nil
			}
		}
	}
}

func getTestComponentID(testComponent *storage.EmbeddedImageScanComponent, imageID string, index int) string {
	return scancomponent.ComponentIDV2(testComponent, imageID, index)
}

func getTestCVEID(testCVE *storage.EmbeddedVulnerability, componentID string, index int) string {
	return cve.IDV2(testCVE, componentID, index)
}

func getTestCVE() *storage.EmbeddedVulnerability {
	parsedTime, _ := time.Parse(time.RFC3339, "2022-05-12T00:00:00Z")

	return &storage.EmbeddedVulnerability{
		Cve:          "CVE-2022-1552",
		Cvss:         8.8,
		Summary:      "DOCUMENTATION: A flaw was found in PostgreSQL. There is an issue with incomplete efforts to operate safely when a privileged user is maintaining another user's objects. The Autovacuum, REINDEX, CREATE INDEX, REFRESH MATERIALIZED VIEW, CLUSTER, and pg_amcheck commands activated relevant protections too late or not at all during the process. This flaw allows an attacker with permission to create non-temporary objects in at least one schema to execute arbitrary SQL functions under a superuser identity.                           MITIGATION: Red Hat has investigated whether a possible mitigation exists for this issue, and has not been able to identify a practical example. Please update the affected package as soon as possible.",
		Link:         "https://access.redhat.com/security/cve/CVE-2022-1552",
		ScoreVersion: storage.EmbeddedVulnerability_V3,
		CvssV3: &storage.CVSSV3{
			Vector:              "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
			ExploitabilityScore: 2.8,
			ImpactScore:         5.9,
			AttackVector:        storage.CVSSV3_ATTACK_NETWORK,
			AttackComplexity:    storage.CVSSV3_COMPLEXITY_LOW,
			PrivilegesRequired:  storage.CVSSV3_PRIVILEGE_LOW,
			UserInteraction:     storage.CVSSV3_UI_NONE,
			Scope:               storage.CVSSV3_UNCHANGED,
			Confidentiality:     storage.CVSSV3_IMPACT_HIGH,
			Integrity:           storage.CVSSV3_IMPACT_HIGH,
			Availability:        storage.CVSSV3_IMPACT_HIGH,
			Score:               8.8,
			Severity:            storage.CVSSV3_HIGH,
		},
		PublishedOn:       timestamppb.New(parsedTime),
		VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		VulnerabilityTypes: []storage.EmbeddedVulnerability_VulnerabilityType{
			storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
		},
		Suppressed: false,
		Severity:   storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
		State:      storage.VulnerabilityState_OBSERVED,
	}
}

func assertResponsesAreEqual(t *testing.T, expected []CveFlat, actual []CveFlat, testOrder bool) {
	if !testOrder {
		sort.SliceStable(expected, func(i, j int) bool {
			return expected[i].GetCVE() < expected[j].GetCVE()
		})
		sort.SliceStable(actual, func(i, j int) bool {
			return actual[i].GetCVE() < actual[j].GetCVE()
		})
	}
	for i, flatCVE := range actual {
		assert.ElementsMatch(t, expected[i].GetCVEIDs(), flatCVE.GetCVEIDs())
		assert.Equal(t, expected[i].GetCVE(), flatCVE.GetCVE())
		assert.Equal(t, expected[i].GetSeverity().String(), flatCVE.GetSeverity().String())
		assert.Equal(t, expected[i].GetTopCVSS(), flatCVE.GetTopCVSS())
		assert.Equal(t, expected[i].GetTopNVDCVSS(), flatCVE.GetTopNVDCVSS())
		assert.Equal(t, expected[i].GetEPSSProbability(), flatCVE.GetEPSSProbability())
		assert.Equal(t, expected[i].GetAffectedImageCount(), flatCVE.GetAffectedImageCount())
		assert.Equal(t, expected[i].GetFirstDiscoveredInSystem(), flatCVE.GetFirstDiscoveredInSystem())
		assert.Equal(t, expected[i].GetPublishDate(), flatCVE.GetPublishDate())
		assert.Equal(t, expected[i].GetFirstImageOccurrence(), flatCVE.GetFirstImageOccurrence())
		assert.Equal(t, expected[i].GetState().String(), flatCVE.GetState().String())
	}
}
