//go:build sql_integration

package imagecveflat

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imagePostgresV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
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
	"go.uber.org/mock/gomock"
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

type filterImpl struct {
	matchImage func(image *storage.Image) bool
	matchVuln  func(vuln *storage.EmbeddedVulnerability) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.Image) bool {
			return true
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.Image) bool {
			return false
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return false
		},
	}
}

func (f *filterImpl) withImageFilter(fn func(image *storage.Image) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withVulnFilter(fn func(vuln *storage.EmbeddedVulnerability) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestImageCVEVFlatiew(t *testing.T) {
	if !features.FlattenCVEData.Enabled() {
		t.Skip("FlattenCVEData is disabled")
	}
	suite.Run(t, new(ImageCVEFlatViewTestSuite))
}

type ImageCVEFlatViewTestSuite struct {
	suite.Suite

	testDB  *pgtest.TestPostgres
	cveView CveFlatView

	testImages              []*storage.Image
	testImagesToDeployments map[string][]*storage.Deployment
}

func (s *ImageCVEFlatViewTestSuite) SetupSuite() {
	mockCtrl := gomock.NewController(s.T())
	ctx := sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	mockRisk := mockRisks.NewMockDataStore(mockCtrl)

	// Initialize the datastore.
	imageStore := imageDS.NewWithPostgres(
		imagePostgresV2.New(s.testDB.DB, false, concurrency.NewKeyFence()),
		mockRisk,
		ranking.ImageRanker(),
		ranking.ComponentRanker(),
	)
	deploymentStore, err := deploymentDS.NewTestDataStore(
		s.T(),
		s.testDB,
		&deploymentDS.DeploymentTestStoreParams{
			ImagesDataStore:  imageStore,
			RisksDataStore:   mockRisk,
			ClusterRanker:    ranking.ClusterRanker(),
			NamespaceRanker:  ranking.NamespaceRanker(),
			DeploymentRanker: ranking.DeploymentRanker(),
		},
	)
	s.Require().NoError(err)

	// Upsert test images.
	images, err := imageSamples.GetTestImages(s.T())
	s.Require().NoError(err)
	// set cvss metrics list with one nvd cvss score
	for _, image := range images {
		for _, components := range image.GetScan().GetComponents() {
			for _, vuln := range components.GetVulns() {
				cvssScore := &storage.CVSSScore{
					Source: storage.Source_SOURCE_NVD,
					CvssScore: &storage.CVSSScore_Cvssv3{
						Cvssv3: &storage.CVSSV3{
							Score: 10,
						},
					},
				}
				vuln.CvssMetrics = []*storage.CVSSScore{cvssScore}
				vuln.NvdCvss = 10
			}
		}
		s.Require().NoError(imageStore.UpsertImage(ctx, image))
	}

	// Ensure that the image is stored and constructed as expected.
	for idx, image := range images {
		actual, found, err := imageStore.GetImage(ctx, image.GetId())
		s.Require().NoError(err)
		s.Require().True(found)

		cloned := actual.CloneVT()
		// Adjust dynamic fields and ensure images in ACS are as expected.
		standardizeImages(image, cloned)

		// Now that we confirmed that images match, use stored image to establish the expected test results.
		// This makes dynamic fields matching (e.g. created at) straightforward.
		images[idx] = actual
	}
	s.testImages = images
	s.cveView = NewCVEFlatView(s.testDB.DB)

	s.Require().Len(images, 5)
	deployments := []*storage.Deployment{
		fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, images[1]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[1]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[2]),
	}
	for _, d := range deployments {
		s.Require().NoError(deploymentStore.UpsertDeployment(ctx, d))
	}

	s.testImagesToDeployments = make(map[string][]*storage.Deployment)
	s.testImagesToDeployments[images[1].Id] = []*storage.Deployment{deployments[0], deployments[1]}
	s.testImagesToDeployments[images[2].Id] = []*storage.Deployment{deployments[2]}
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

			expected := compileExpected(s.testImages, tc.matchFilter, tc.readOptions, tc.less)
			assert.Equal(t, len(expected), len(actual))
			assert.ElementsMatch(t, expected, actual)
			if tc.testOrder {
				assert.Equal(t, expected, actual)
			}

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
				matchFilter.withImageFilter(func(image *storage.Image) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expected := compileExpected(s.testImages, &matchFilter, tc.readOptions, tc.less)
				assert.Equal(t, len(expected), len(actual))
				assert.ElementsMatch(t, expected, actual)
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
		assert.Equal(t, []CveFlat{}, actual)
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

				expected := compileExpected(s.testImages, tc.matchFilter, tc.readOptions, tc.less)

				assert.Equal(t, len(expected), len(actual))
				assert.EqualValues(t, expected, actual)

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

			expected := compileExpected(s.testImages, tc.matchFilter, tc.readOptions, nil)
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
				matchFilter.withImageFilter(func(image *storage.Image) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expected := compileExpected(s.testImages, &matchFilter, tc.readOptions, tc.less)
				assert.Equal(t, len(expected), actual)
			})
		}
	}
}

func (s *ImageCVEFlatViewTestSuite) testCases() []testCase {
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
			matchFilter: matchAllFilter().withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
				return vuln.GetCve() == "CVE-2022-1552"
			}),
		},
		{
			desc: "search one image",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:latest").ProtoQuery(),
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
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
				withImageFilter(func(image *storage.Image) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
				}).
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2022-1552"
				}),
		},
		{
			desc: "search critical severity",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
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
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
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
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2015-8704" && vuln.GetFixedBy() != ""
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
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2015-8704" && vuln.GetFixedBy() == ""
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
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
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
				withImageFilter(func(image *storage.Image) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
				}).
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
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
				IDs:   []string{"sha256:6ef31316f4f9e0c31a8f4e602ba287a210d66934f91b1616f1c9b957201d025c"},
				Level: v1.SearchCategory_IMAGES,
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.CVE, "CVE-2022-1552").
				AddExactMatches(search.ImageName, "quay.io/appcontainers/wordpress:debian").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image *storage.Image) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian"
				}).
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2022-1552"
				}),
		},
		{
			desc: "search critical severity w/ cve & image scope",
			ctx: scoped.Context(context.Background(), scoped.Scope{
				IDs:   []string{"sha256:6ef31316f4f9e0c31a8f4e602ba287a210d66934f91b1616f1c9b957201d025c"},
				Level: v1.SearchCategory_IMAGES,
				Parent: &scoped.Scope{
					IDs: []string{cve.IDV2("CVE-2022-1552", scancomponent.ComponentIDV2(&storage.EmbeddedImageScanComponent{
						Name:         "postgresql-libs",
						Version:      "8.4.20-6.el6",
						Source:       storage.SourceType_OS,
						Location:     "",
						Architecture: "",
					}, "sha256:05dd8ed5c76ad3c9f06481770828cf17b8c89f1e406c91d548426dd70fe94560"), "20")},
					Level: v1.SearchCategory_IMAGE_VULNERABILITIES,
				},
			}),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Severity, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY.String()).
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image *storage.Image) bool {
					return image.GetName().GetFullName() == "quay.io/appcontainers/wordpress:debian" &&
						image.GetScan().GetOperatingSystem() == "debian:8"
				}).
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.GetCve() == "CVE-2022-1552" &&
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
				withImageFilter(func(image *storage.Image) bool {
					deps, ok := s.testImagesToDeployments[image.GetId()]
					if !ok {
						// include inactive image
						return true
					}
					for _, d := range deps {
						if !d.PlatformComponent {
							return true
						}
					}
					return false
				}).
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.State == storage.VulnerabilityState_OBSERVED
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
				withImageFilter(func(image *storage.Image) bool {
					deps, ok := s.testImagesToDeployments[image.GetId()]
					if !ok {
						// include inactive image
						return true
					}
					for _, d := range deps {
						if d.PlatformComponent {
							return true
						}
					}
					return false
				}).
				withVulnFilter(func(vuln *storage.EmbeddedVulnerability) bool {
					return vuln.State == storage.VulnerabilityState_OBSERVED
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
					search.NewSortOption(search.ImageSHA).AggregateBy(aggregatefunc.Count, true).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			less: func(records []*imageCVEFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					if records[i].AffectedImageCount == records[j].AffectedImageCount {
						return records[i].CVE < records[j].CVE
					}
					return records[i].AffectedImageCount > records[j].AffectedImageCount
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

func compileExpected(images []*storage.Image, filter *filterImpl, options views.ReadOptions, less lessFunc) []CveFlat {
	cveMap := make(map[string]*imageCVEFlatResponse)

	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		var seenForImage set.Set[string]
		for _, component := range image.GetScan().GetComponents() {
			for vulnIdx, vuln := range component.GetVulns() {
				if !filter.matchVuln(vuln) {
					continue
				}

				vulnTime, _ := protocompat.ConvertTimestampToTimeOrError(vuln.GetFirstSystemOccurrence())
				vulnTime = vulnTime.Round(time.Microsecond)
				vulnPublishDate, _ := protocompat.ConvertTimestampToTimeOrError(vuln.GetPublishedOn())
				vulnPublishDate = vulnPublishDate.Round(time.Microsecond)
				vulnImageOccurrence, _ := protocompat.ConvertTimestampToTimeOrError(vuln.GetFirstImageOccurrence())
				vulnImageOccurrence = vulnImageOccurrence.Round(time.Microsecond)
				val := cveMap[vuln.GetCve()]

				var impactScore float32
				if vuln.GetCvssV3() != nil {
					impactScore = vuln.GetCvssV3().GetImpactScore()
				} else if vuln.GetCvssV2() != nil {
					impactScore = vuln.GetCvssV2().GetImpactScore()
				}

				if val == nil {
					val = &imageCVEFlatResponse{
						CVE:                     vuln.GetCve(),
						TopCVSS:                 pointers.Float32(vuln.GetCvss()),
						FirstDiscoveredInSystem: &vulnTime,
						Published:               &vulnPublishDate,
						Severity:                vuln.GetSeverity().Enum(),
						FirstImageOccurrence:    &vulnImageOccurrence,
						State:                   pointers.Pointer(vuln.GetState()),
						ImpactScore:             pointers.Float32(impactScore),
						EpssProbability:         pointers.Float32(vuln.GetEpss().GetEpssProbability()),
					}
					for _, metric := range vuln.CvssMetrics {
						if metric.Source == storage.Source_SOURCE_NVD {
							if metric.GetCvssv2() != nil {
								val.TopNVDCVSS = pointers.Float32(metric.GetCvssv2().GetScore())
							} else {
								val.TopNVDCVSS = pointers.Float32(metric.GetCvssv3().GetScore())
							}
						}
					}
					cveMap[val.CVE] = val
				}

				val.TopCVSS = pointers.Float32(max(val.GetTopCVSS(), vuln.GetCvss()))
				val.ImpactScore = pointers.Float32(max(*val.ImpactScore, impactScore))
				val.EpssProbability = pointers.Float32(max(*val.EpssProbability, vuln.GetEpss().GetEpssProbability()))
				if vuln.GetSeverity().Number() > val.GetSeverity().Number() {
					val.Severity = pointers.Pointer(vuln.GetSeverity())
				}

				id := cve.IDV2(val.GetCVE(), scancomponent.ComponentIDV2(component, image.GetId()), strconv.Itoa(vulnIdx))
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
				val.AffectedImageCount++
			}
		}
	}

	expected := make([]*imageCVEFlatResponse, 0, len(cveMap))
	for _, entry := range cveMap {
		sort.SliceStable(entry.CVEIDs, func(i, j int) bool {
			return entry.CVEIDs[i] < entry.CVEIDs[j]
		})
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
