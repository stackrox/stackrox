//go:build sql_integration

package imagecve

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/views"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/cve"
	"github.com/stackrox/rox/pkg/fixtures"
	imageSamples "github.com/stackrox/rox/pkg/fixtures/image"
	"github.com/stackrox/rox/pkg/mathutil"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type testCase struct {
	desc        string
	ctx         context.Context
	q           *v1.Query
	matchFilter *filterImpl
	less        lessFunc
	readOptions views.ReadOptions
	expectedErr string
}

type lessFunc func(records []*imageCVECoreResponse) func(i, j int) bool

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

func TestImageCVEView(t *testing.T) {
	suite.Run(t, new(ImageCVEViewTestSuite))
}

type ImageCVEViewTestSuite struct {
	suite.Suite

	testDB  *pgtest.TestPostgres
	cveView CveView

	testImages []*storage.Image
}

func (s *ImageCVEViewTestSuite) SetupSuite() {

	ctx := sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Initialize the datastore.
	imageStore := imageDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	// Upsert test images.
	images, err := imageSamples.GetTestImages(s.T())
	s.Require().NoError(err)
	for _, image := range images {
		s.Require().NoError(imageStore.UpsertImage(ctx, image))
	}

	// Ensure that the image is stored and constructed as expected.
	for idx, image := range images {
		actual, found, err := imageStore.GetImage(ctx, image.GetId())
		s.Require().NoError(err)
		s.Require().True(found)

		cloned := actual.Clone()
		// Adjust dynamic fields and ensure images in ACS are as expected.
		standardizeImages(image, cloned)
		s.Require().EqualValues(image, cloned)

		// Now that we confirmed that images match, use stored image to establish the expected test results.
		// This makes dynamic fields matching (e.g. created at) straightforward.
		images[idx] = actual
	}
	s.testImages = images
	s.cveView = NewCVEView(s.testDB.DB)

	s.Require().Len(images, 5)
	deployments := []*storage.Deployment{
		fixtures.GetDeploymentWithImage(testconsts.Cluster1, testconsts.NamespaceA, images[1]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[1]),
		fixtures.GetDeploymentWithImage(testconsts.Cluster2, testconsts.NamespaceB, images[2]),
	}
	for _, d := range deployments {
		s.Require().NoError(deploymentStore.UpsertDeployment(ctx, d))
	}
}

func (s *ImageCVEViewTestSuite) TearDownSuite() {
	s.testDB.Teardown(s.T())
}

func (s *ImageCVEViewTestSuite) TestGetImageCVECore() {
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

			if tc.readOptions.SkipGetAffectedImages || tc.readOptions.SkipGetImagesBySeverity {
				return
			}

			for _, record := range actual {
				// The total count should be equal to aggregation of the all severity counts.
				assert.Equal(t,
					record.GetImagesBySeverity().GetLowSeverityCount().GetTotal()+
						record.GetImagesBySeverity().GetModerateSeverityCount().GetTotal()+
						record.GetImagesBySeverity().GetImportantSeverityCount().GetTotal()+
						record.GetImagesBySeverity().GetCriticalSeverityCount().GetTotal(),
					record.GetAffectedImageCount(),
				)

			}
		})
	}
}

func (s *ImageCVEViewTestSuite) TestGetImageCVECoreSAC() {
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
				matchFilter := tc.matchFilter
				baseImageMatchFilter := matchFilter.matchImage
				matchFilter.withImageFilter(func(image *storage.Image) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expected := compileExpected(s.testImages, matchFilter, tc.readOptions, tc.less)
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
		assert.Equal(t, []CveCore{}, actual)
	})
}

func (s *ImageCVEViewTestSuite) TestGetImageIDs() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			// Such testcases are meant only for Get().
			if tc.expectedErr != "" {
				return
			}

			query := tc.q.Clone()
			query.Pagination = nil
			actualAffectedImageIDs, err := s.cveView.GetImageIDs(sac.WithAllAccess(tc.ctx), query)
			assert.NoError(t, err)
			expectedAffectedImages := compileExpectedAffectedImageIDs(s.testImages, tc.matchFilter)
			assert.ElementsMatch(t, expectedAffectedImages, actualAffectedImageIDs)
		})
	}
}

func (s *ImageCVEViewTestSuite) TestGetImageIDsSAC() {
	for _, tc := range s.testCases() {
		for key, sacTC := range s.sacTestCases() {
			s.T().Run(fmt.Sprintf("Image %s %s", key, tc.desc), func(t *testing.T) {
				// Such testcases are meant only for Get().
				if tc.expectedErr != "" {
					return
				}

				testCtxs := testutils.GetNamespaceScopedTestContexts(tc.ctx, s.T(), resources.Image)
				ctx := testCtxs[key]
				query := tc.q.Clone()
				query.Pagination = nil
				actualAffectedImageIDs, err := s.cveView.GetImageIDs(ctx, query)
				assert.NoError(t, err)

				// Wrap image filter with sac filter.
				matchFilter := tc.matchFilter
				baseImageMatchFilter := matchFilter.matchImage
				matchFilter.withImageFilter(func(image *storage.Image) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expectedAffectedImages := compileExpectedAffectedImageIDs(s.testImages, tc.matchFilter)
				assert.ElementsMatch(t, expectedAffectedImages, actualAffectedImageIDs)
			})
		}
	}
}

func (s *ImageCVEViewTestSuite) TestGetImageCVECoreWithPagination() {
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

				for _, record := range actual {
					assert.Equal(t,
						record.GetImagesBySeverity().GetLowSeverityCount().GetTotal()+
							record.GetImagesBySeverity().GetModerateSeverityCount().GetTotal()+
							record.GetImagesBySeverity().GetImportantSeverityCount().GetTotal()+
							record.GetImagesBySeverity().GetCriticalSeverityCount().GetTotal(),
						record.GetAffectedImageCount(),
					)
				}
			})
		}
	}
}

func (s *ImageCVEViewTestSuite) TestCountImageCVECore() {
	for _, tc := range s.testCases() {
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

func (s *ImageCVEViewTestSuite) TestCountImageCVECoreSAC() {
	for _, tc := range s.testCases() {
		for key, sacTC := range s.sacTestCases() {
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
				matchFilter := tc.matchFilter
				baseImageMatchFilter := matchFilter.matchImage
				matchFilter.withImageFilter(func(image *storage.Image) bool {
					if sacTC[image.GetId()] {
						return baseImageMatchFilter(image)
					}
					return false
				})

				expected := compileExpected(s.testImages, matchFilter, tc.readOptions, tc.less)
				assert.Equal(t, len(expected), actual)
			})
		}
	}
}

func (s *ImageCVEViewTestSuite) TestCountBySeverity() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.cveView.CountBySeverity(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := compileExpectedCountBySeverity(s.testImages, tc.matchFilter)
			assert.EqualValues(t, expected, actual)
		})
	}
}

func (s *ImageCVEViewTestSuite) testCases() []testCase {
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
			desc: "search one operating system",
			ctx:  context.Background(),
			q:    search.NewQueryBuilder().AddExactMatches(search.OperatingSystem, "debian:8").ProtoQuery(),
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.Image) bool {
				return image.GetScan().GetOperatingSystem() == "debian:8"
			}),
		},
		{
			desc:        "no match",
			ctx:         context.Background(),
			q:           search.NewQueryBuilder().AddExactMatches(search.OperatingSystem, "").ProtoQuery(),
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
				ID:    "sha256:6ef31316f4f9e0c31a8f4e602ba287a210d66934f91b1616f1c9b957201d025c",
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
				ID:    "sha256:6ef31316f4f9e0c31a8f4e602ba287a210d66934f91b1616f1c9b957201d025c",
				Level: v1.SearchCategory_IMAGES,
				Parent: &scoped.Scope{
					ID:    cve.ID("CVE-2022-1552", "debian:8"),
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
	}
}

func (s *ImageCVEViewTestSuite) paginationTestCases() []testCase {
	return []testCase{
		{
			desc: "w/ affected image sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.ImageSHA).AggregateBy(aggregatefunc.Count, true).Reversed(true),
				).AddSortOption(search.NewSortOption(search.CVE)),
			).ProtoQuery(),
			less: func(records []*imageCVECoreResponse) func(i, j int) bool {
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
			less: func(records []*imageCVECoreResponse) func(i, j int) bool {
				return func(i, j int) bool {
					if records[i].TopCVSS == records[j].TopCVSS {
						return records[i].CVE < records[j].CVE
					}
					return records[i].TopCVSS > records[j].TopCVSS
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
			less: func(records []*imageCVECoreResponse) func(i, j int) bool {
				return func(i, j int) bool {
					recordI, recordJ := records[i], records[j]
					if recordJ == nil {
						recordJ = &imageCVECoreResponse{}
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

func (s *ImageCVEViewTestSuite) sacTestCases() map[string]map[string]bool {
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

func compileExpected(images []*storage.Image, filter *filterImpl, options views.ReadOptions, less lessFunc) []CveCore {
	cveMap := make(map[string]*imageCVECoreResponse)

	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		var seenForImage set.Set[string]
		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				if !filter.matchVuln(vuln) {
					continue
				}

				vulnTime, _ := types.TimestampFromProto(vuln.GetFirstSystemOccurrence())
				vulnTime = vulnTime.Round(time.Microsecond)
				val := cveMap[vuln.GetCve()]
				if val == nil {
					val = &imageCVECoreResponse{
						CVE:                     vuln.GetCve(),
						TopCVSS:                 vuln.GetCvss(),
						FirstDiscoveredInSystem: &vulnTime,
					}
					cveMap[val.CVE] = val
				}

				val.TopCVSS = mathutil.MaxFloat32(val.GetTopCVSS(), vuln.GetCvss())

				id := cve.ID(val.GetCVE(), image.GetScan().GetOperatingSystem())
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

				if !seenForImage.Add(val.CVE) {
					continue
				}
				val.AffectedImageCount++

				switch vuln.GetSeverity() {
				case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
					val.ImagesWithCriticalSeverity++
					if vuln.GetFixedBy() != "" {
						val.FixableImagesWithCriticalSeverity++
					}
				case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
					val.ImagesWithImportantSeverity++
					if vuln.GetFixedBy() != "" {
						val.FixableImagesWithImportantSeverity++
					}
				case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
					val.ImagesWithModerateSeverity++
					if vuln.GetFixedBy() != "" {
						val.FixableImagesWithModerateSeverity++
					}
				case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
					val.ImagesWithLowSeverity++
					if vuln.GetFixedBy() != "" {
						val.FixableImagesWithLowSeverity++
					}
				}
			}
		}
	}

	expected := make([]*imageCVECoreResponse, 0, len(cveMap))
	for _, entry := range cveMap {
		sort.SliceStable(entry.CVEIDs, func(i, j int) bool {
			return entry.CVEIDs[i] < entry.CVEIDs[j]
		})
		expected = append(expected, entry)
	}
	if options.SkipGetImagesBySeverity {
		for _, entry := range expected {
			entry.ImagesWithLowSeverity = 0
			entry.FixableImagesWithLowSeverity = 0

			entry.ImagesWithModerateSeverity = 0
			entry.FixableImagesWithModerateSeverity = 0

			entry.ImagesWithImportantSeverity = 0
			entry.FixableImagesWithImportantSeverity = 0

			entry.ImagesWithCriticalSeverity = 0
			entry.FixableImagesWithCriticalSeverity = 0
		}
	}
	if options.SkipGetTopCVSS {
		for _, entry := range expected {
			entry.TopCVSS = 0
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

	ret := make([]CveCore, 0, len(cveMap))
	for _, entry := range expected {
		ret = append(ret, entry)
	}
	return ret
}

func compileExpectedAffectedImageIDs(images []*storage.Image, filter *filterImpl) []string {
	var affectedImageIDs []string
	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		for _, component := range image.GetScan().GetComponents() {
			var vulnFilterPassed bool
			for _, vuln := range component.GetVulns() {
				if filter.matchVuln(vuln) {
					vulnFilterPassed = true
					break
				}
			}
			if vulnFilterPassed {
				affectedImageIDs = append(affectedImageIDs, image.GetId())
				break
			}
		}
	}
	return affectedImageIDs
}

func compileExpectedCountBySeverity(images []*storage.Image, filter *filterImpl) *resourceCountByImageCVESeverity {
	sevToCVEsMap := make(map[storage.VulnerabilitySeverity]set.Set[string])
	sevToFixableCVEsMap := make(map[storage.VulnerabilitySeverity]set.Set[string])

	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		for _, component := range image.GetScan().GetComponents() {
			for _, vuln := range component.GetVulns() {
				if !filter.matchVuln(vuln) {
					continue
				}

				if vuln.GetSeverity() == storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY {
					continue
				}
				cves := sevToCVEsMap[vuln.GetSeverity()]
				cves.Add(vuln.GetCve())
				sevToCVEsMap[vuln.GetSeverity()] = cves

				if vuln.GetFixedBy() != "" {
					cves = sevToFixableCVEsMap[vuln.GetSeverity()]
					cves.Add(vuln.GetCve())
					sevToFixableCVEsMap[vuln.GetSeverity()] = cves
				}
			}
		}
	}
	return &resourceCountByImageCVESeverity{
		CriticalSeverityCount:        sevToCVEsMap[storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY].Cardinality(),
		FixableCriticalSeverityCount: sevToFixableCVEsMap[storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY].Cardinality(),

		ImportantSeverityCount:        sevToCVEsMap[storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY].Cardinality(),
		FixableImportantSeverityCount: sevToFixableCVEsMap[storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY].Cardinality(),

		ModerateSeverityCount:        sevToCVEsMap[storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY].Cardinality(),
		FixableModerateSeverityCount: sevToFixableCVEsMap[storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY].Cardinality(),

		LowSeverityCount:        sevToCVEsMap[storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY].Cardinality(),
		FixableLowSeverityCount: sevToFixableCVEsMap[storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY].Cardinality(),
	}
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

			sort.SliceStable(vulns, func(i, j int) bool {
				return vulns[i].Cve < vulns[j].Cve
			})
		}

		sort.SliceStable(components, func(i, j int) bool {
			if components[i].Name == components[j].Name {
				return components[i].Version < components[j].Version
			}
			return components[i].Name < components[j].Name
		})
	}
}
