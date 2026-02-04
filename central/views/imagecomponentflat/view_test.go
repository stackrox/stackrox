//go:build sql_integration

package imagecomponentflat

import (
	"context"
	"fmt"
	"sort"
	"testing"

	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/aggregatefunc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type testCase struct {
	desc        string
	ctx         context.Context
	q           *v1.Query
	matchFilter *filterImpl
	less        lessFunc
	expectedErr string
}

type lessFunc func(records []*imageComponentFlatResponse) func(i, j int) bool

type filterImpl struct {
	matchImage     func(image *storage.ImageV2) bool
	matchComponent func(component *storage.EmbeddedImageScanComponent) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.ImageV2) bool {
			return true
		},
		matchComponent: func(_ *storage.EmbeddedImageScanComponent) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.ImageV2) bool {
			return false
		},
		matchComponent: func(_ *storage.EmbeddedImageScanComponent) bool {
			return false
		},
	}
}

func (f *filterImpl) withImageFilter(fn func(image *storage.ImageV2) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withComponentFilter(fn func(component *storage.EmbeddedImageScanComponent) bool) *filterImpl {
	f.matchComponent = fn
	return f
}

func TestImageComponentFlatView(t *testing.T) {
	if !features.FlattenImageData.Enabled() {
		t.Skip("FlattenImageData is disabled")
	}

	suite.Run(t, new(ImageComponentFlatViewTestSuite))
}

type ImageComponentFlatViewTestSuite struct {
	suite.Suite

	testDB         *pgtest.TestPostgres
	componentView  ComponentFlatView
	testImages     []*storage.ImageV2
	testComponents map[string][]*storage.EmbeddedImageScanComponent
}

func (s *ImageComponentFlatViewTestSuite) SetupSuite() {
	ctx := sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())

	// Initialize the ImageV2 datastore
	imageStore := imageV2DS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	deploymentStore, err := deploymentDS.GetTestPostgresDataStore(s.T(), s.testDB.DB)
	s.Require().NoError(err)

	// Create controlled test images with known components
	images := s.createTestImages()

	// Upsert images
	for _, image := range images {
		s.Require().NoError(imageStore.UpsertImage(ctx, image))
	}

	// Ensure that the images are stored and constructed as expected.
	for idx, image := range images {
		actual, found, err := imageStore.GetImage(ctx, image.GetId())
		s.Require().NoError(err)
		s.Require().True(found)

		// Use stored image to establish the expected test results.
		images[idx] = actual
	}
	s.testImages = images
	s.componentView = NewComponentFlatView(s.testDB.DB)

	// Build component mapping for test validation
	s.testComponents = make(map[string][]*storage.EmbeddedImageScanComponent)
	for _, image := range s.testImages {
		s.testComponents[image.GetId()] = image.GetScan().GetComponents()
	}

	// Create some deployments for testing
	s.Require().Len(images, 3)
	deployments := []*storage.Deployment{
		fixtures.GetDeploymentWithImageV2(testconsts.Cluster1, testconsts.NamespaceA, images[0]),
		fixtures.GetDeploymentWithImageV2(testconsts.Cluster2, testconsts.NamespaceB, images[1]),
		fixtures.GetDeploymentWithImageV2(testconsts.Cluster2, testconsts.NamespaceB, images[2]),
	}
	for _, d := range deployments {
		s.Require().NoError(deploymentStore.UpsertDeployment(ctx, d))
	}
}

func (s *ImageComponentFlatViewTestSuite) TestGetImageComponentFlat() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.componentView.Get(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpected(s.testImages, tc.matchFilter, tc.less)
			assert.Equal(t, len(expected), len(actual))
			s.assertComponentResponsesAreEqual(t, expected, actual, tc.less != nil)
		})
	}
}

func (s *ImageComponentFlatViewTestSuite) TestGetImageComponentFlatSAC() {
	for _, tc := range s.testCases() {
		for key := range s.sacTestCases() {
			s.T().Run(fmt.Sprintf("Image %s %s", key, tc.desc), func(t *testing.T) {
				testCtxs := testutils.GetNamespaceScopedTestContexts(tc.ctx, s.T(), resources.Image)
				ctx := testCtxs[key]

				actual, err := s.componentView.Get(ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Compute expected results based on SAC filtering
				expected := s.compileExpectedWithSAC(key, tc.matchFilter, tc.less)

				// Validate exact results - we can predict SAC filtering because we know the restrictions
				assert.Equal(t, len(expected), len(actual), "SAC filtering should produce predictable results for %s", key)
				s.assertComponentResponsesAreEqual(t, expected, actual, tc.less != nil)
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

		actual, err := s.componentView.Get(ctx, tc.q)
		if tc.expectedErr != "" {
			s.ErrorContains(err, tc.expectedErr)
			return
		}
		assert.NoError(t, err)
		// For deployment access test, we expect no results since we only have image access
		assert.Empty(t, actual, "Expected no results when querying with deployment access but only having image access")
	})
}

func (s *ImageComponentFlatViewTestSuite) TestGetImageComponentFlatWithPagination() {
	for _, paginationTestCase := range s.paginationTestCases() {
		baseTestCases := s.testCases()
		for idx := range baseTestCases {
			tc := &baseTestCases[idx]
			applyPaginationProps(tc, paginationTestCase)

			s.T().Run(tc.desc, func(t *testing.T) {
				actual, err := s.componentView.Get(sac.WithAllAccess(tc.ctx), tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// For pagination tests, validate that results are properly structured
				// and that pagination is working (different page sizes should return different counts)
				expected := s.compileExpected(s.testImages, tc.matchFilter, tc.less)

				// Additional pagination validation: if we have pagination, results should be limited
				if tc.q.GetPagination() != nil && tc.q.GetPagination().GetLimit() > 0 {
					assert.LessOrEqual(t, len(actual), int(tc.q.GetPagination().GetLimit()),
						"Paginated results should not exceed the specified limit")
					// For paginated results, we can't compare exact data since pagination affects the results
					// Just validate structure
					for i, result := range actual {
						assert.NotEmpty(t, result.GetComponent(), "Component name should not be empty for result %d", i)
						assert.NotEmpty(t, result.GetComponentIDs(), "Component IDs should not be empty for result %d", i)
					}
				} else {
					// No pagination, so we can compare exact results
					assert.Equal(t, len(expected), len(actual))
					s.assertComponentResponsesAreEqual(t, expected, actual, true)
				}
			})
		}
	}
}

func (s *ImageComponentFlatViewTestSuite) TestCountImageComponentFlat() {
	for _, tc := range s.testCases() {
		s.T().Run(tc.desc, func(t *testing.T) {
			actual, err := s.componentView.Count(sac.WithAllAccess(tc.ctx), tc.q)
			if tc.expectedErr != "" {
				s.ErrorContains(err, tc.expectedErr)
				return
			}
			assert.NoError(t, err)

			expected := s.compileExpected(s.testImages, tc.matchFilter, nil)
			assert.Equal(t, len(expected), actual)
		})
	}
}

func (s *ImageComponentFlatViewTestSuite) TestCountImageComponentFlatSAC() {
	for _, tc := range s.testCases() {
		for key := range s.sacTestCases() {
			s.T().Run(fmt.Sprintf("Image %s %s", key, tc.desc), func(t *testing.T) {
				testCtxs := testutils.GetNamespaceScopedTestContexts(tc.ctx, s.T(), resources.Image)
				ctx := testCtxs[key]

				actual, err := s.componentView.Count(ctx, tc.q)
				if tc.expectedErr != "" {
					s.ErrorContains(err, tc.expectedErr)
					return
				}
				assert.NoError(t, err)

				// Compute expected count based on SAC filtering
				expected := s.compileExpectedWithSAC(key, tc.matchFilter, nil)
				expectedCount := len(expected)

				// Validate exact count - we can predict SAC filtering because we know the restrictions
				assert.Equal(t, expectedCount, actual, "SAC filtering should produce predictable count for %s", key)
			})
		}
	}
}

// createTestImages creates a small, controlled set of test images with known components
func (s *ImageComponentFlatViewTestSuite) createTestImages() []*storage.ImageV2 {
	return []*storage.ImageV2{
		// Image 1: Ubuntu with bash and openssl
		{
			Id:     uuid.NewV5FromNonUUIDs("docker.io/library/ubuntu:20.04", "sha256:ubuntu1").String(),
			Digest: "sha256:ubuntu1",
			Name: &storage.ImageName{
				Registry: "docker.io",
				Remote:   "library/ubuntu",
				Tag:      "20.04",
				FullName: "docker.io/library/ubuntu:20.04",
			},
			Scan: &storage.ImageScan{
				ScanTime:        protocompat.TimestampNow(),
				OperatingSystem: "ubuntu:20.04",
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:       "bash",
						Version:    "5.0-6ubuntu1.2",
						SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 7.5},
						RiskScore:  2.5,
					},
					{
						Name:       "openssl",
						Version:    "1.1.1f-1ubuntu2.16",
						SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 9.8},
						RiskScore:  3.2,
					},
				},
			},
		},
		// Image 2: Debian with curl and nginx
		{
			Id:     uuid.NewV5FromNonUUIDs("docker.io/library/debian:bullseye", "sha256:debian1").String(),
			Digest: "sha256:debian1",
			Name: &storage.ImageName{
				Registry: "docker.io",
				Remote:   "library/debian",
				Tag:      "bullseye",
				FullName: "docker.io/library/debian:bullseye",
			},
			Scan: &storage.ImageScan{
				ScanTime:        protocompat.TimestampNow(),
				OperatingSystem: "debian:bullseye",
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:       "curl",
						Version:    "7.74.0-1.3+deb11u2",
						SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 6.5},
						RiskScore:  2.1,
					},
					{
						Name:       "nginx",
						Version:    "1.18.0-6.1+deb11u3",
						SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 5.3},
						RiskScore:  1.8,
					},
				},
			},
		},
		// Image 3: Alpine with wget
		{
			Id:     uuid.NewV5FromNonUUIDs("docker.io/library/alpine:3.16", "sha256:alpine1").String(),
			Digest: "sha256:alpine1",
			Name: &storage.ImageName{
				Registry: "docker.io",
				Remote:   "library/alpine",
				Tag:      "3.16",
				FullName: "docker.io/library/alpine:3.16",
			},
			Scan: &storage.ImageScan{
				ScanTime:        protocompat.TimestampNow(),
				OperatingSystem: "alpine:3.16",
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:       "wget",
						Version:    "1.21.3-r0",
						SetTopCvss: &storage.EmbeddedImageScanComponent_TopCvss{TopCvss: 8.1},
						RiskScore:  2.8,
					},
				},
			},
		},
	}
}

func (s *ImageComponentFlatViewTestSuite) testCases() []testCase {
	return []testCase{
		{
			desc:        "search all",
			ctx:         context.Background(),
			q:           search.EmptyQuery(),
			matchFilter: matchAllFilter(),
		},
		{
			desc: "search one component",
			ctx:  context.Background(),
			q:    search.NewQueryBuilder().AddExactMatches(search.Component, "bash").ProtoQuery(),
			matchFilter: matchAllFilter().withComponentFilter(func(component *storage.EmbeddedImageScanComponent) bool {
				return component.GetName() == "bash"
			}),
		},
		{
			desc: "search one image",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ImageName, "docker.io/library/ubuntu:20.04").ProtoQuery(),
			matchFilter: matchAllFilter().withImageFilter(func(image *storage.ImageV2) bool {
				return image.GetName().GetFullName() == "docker.io/library/ubuntu:20.04"
			}),
		},
		{
			desc: "search one component + one image",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.Component, "curl").
				AddExactMatches(search.ImageName, "docker.io/library/debian:bullseye").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image *storage.ImageV2) bool {
					return image.GetName().GetFullName() == "docker.io/library/debian:bullseye"
				}).
				withComponentFilter(func(component *storage.EmbeddedImageScanComponent) bool {
					return component.GetName() == "curl"
				}),
		},
		{
			desc: "search component version",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.ComponentVersion, "1.21.3-r0").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withComponentFilter(func(component *storage.EmbeddedImageScanComponent) bool {
					return component.GetVersion() == "1.21.3-r0"
				}),
		},
		{
			desc: "search operating system",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.OperatingSystem, "debian:bullseye").
				ProtoQuery(),
			matchFilter: matchAllFilter().
				withImageFilter(func(image *storage.ImageV2) bool {
					return image.GetScan().GetOperatingSystem() == "debian:bullseye"
				}),
		},
		{
			desc:        "no match",
			ctx:         context.Background(),
			q:           search.NewQueryBuilder().AddExactMatches(search.Component, "nonexistent-component").ProtoQuery(),
			matchFilter: matchNoneFilter(),
		},
		{
			desc: "with select",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddSelectFields(search.NewQuerySelect(search.Component)).
				AddExactMatches(search.OperatingSystem, "").ProtoQuery(),
			expectedErr: "Unexpected select clause in query",
		},
		{
			desc: "with group by",
			ctx:  context.Background(),
			q: search.NewQueryBuilder().
				AddExactMatches(search.OperatingSystem, "").
				AddGroupBy(search.Component).ProtoQuery(),
			expectedErr: "Unexpected group by clause in query",
		},
	}
}

func (s *ImageComponentFlatViewTestSuite) paginationTestCases() []testCase {
	return []testCase{
		{
			desc: "w/ component name sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.Component),
				),
			).ProtoQuery(),
			less: func(records []*imageComponentFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					return records[i].Component < records[j].Component
				}
			},
		},
		{
			desc: "w/ top cvss sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.ComponentTopCVSS).AggregateBy(aggregatefunc.Max, false).Reversed(true),
				).AddSortOption(search.NewSortOption(search.Component)),
			).ProtoQuery(),
			less: func(records []*imageComponentFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					if records[i].GetTopCVSS() == records[j].GetTopCVSS() {
						return records[i].Component < records[j].Component
					}
					return records[i].GetTopCVSS() > records[j].GetTopCVSS()
				}
			},
		},
		{
			desc: "w/ risk score sort",
			q: search.NewQueryBuilder().WithPagination(
				search.NewPagination().AddSortOption(
					search.NewSortOption(search.ComponentRiskScore).AggregateBy(aggregatefunc.Max, false).Reversed(true),
				).AddSortOption(search.NewSortOption(search.Component)),
			).ProtoQuery(),
			less: func(records []*imageComponentFlatResponse) func(i, j int) bool {
				return func(i, j int) bool {
					if records[i].GetRiskScore() == records[j].GetRiskScore() {
						return records[i].Component < records[j].Component
					}
					return records[i].GetRiskScore() > records[j].GetRiskScore()
				}
			},
		},
	}
}

func (s *ImageComponentFlatViewTestSuite) sacTestCases() map[string]map[string]bool {
	s.Require().Len(s.testImages, 3)

	img1 := s.testImages[0]
	img2 := s.testImages[1]
	img3 := s.testImages[2]

	// The map structure is the mapping ScopeKey -> ImageID -> Visible
	return map[string]map[string]bool{
		testutils.UnrestrictedReadCtx: {
			img1.GetId(): true,
			img2.GetId(): true,
			img3.GetId(): true,
		},
		testutils.UnrestrictedReadWriteCtx: {
			img1.GetId(): true,
			img2.GetId(): true,
			img3.GetId(): true,
		},
		testutils.Cluster1ReadWriteCtx: {
			img1.GetId(): true, // Ubuntu image is deployed in Cluster1
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespaceAReadWriteCtx: {
			img1.GetId(): true, // Ubuntu image is deployed in Cluster1/NamespaceA
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespaceBReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespacesABReadWriteCtx: {
			img1.GetId(): true, // Ubuntu image is in NamespaceA
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster1NamespacesBCReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster2ReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true, // Debian image is deployed in Cluster2
			img3.GetId(): true, // Alpine image is deployed in Cluster2
		},
		testutils.Cluster2NamespaceAReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster2NamespaceBReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true, // Debian image is deployed in Cluster2/NamespaceB
			img3.GetId(): true, // Alpine image is deployed in Cluster2/NamespaceB
		},
		testutils.Cluster2NamespacesACReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): false,
			img3.GetId(): false,
		},
		testutils.Cluster2NamespacesBCReadWriteCtx: {
			img1.GetId(): false,
			img2.GetId(): true, // Debian image is in NamespaceB
			img3.GetId(): true, // Alpine image is in NamespaceB
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

// compileExpected builds the expected results by replicating the view's aggregation logic
func (s *ImageComponentFlatViewTestSuite) compileExpected(images []*storage.ImageV2, filter *filterImpl, less lessFunc) []ComponentFlat {
	// Build expected results by iterating through images and their components,
	// then aggregating according to the view's grouping logic: (component name, version, OS)
	componentMap := make(map[string]*imageComponentFlatResponse)

	for _, image := range images {
		if !filter.matchImage(image) {
			continue
		}

		for idx, component := range image.GetScan().GetComponents() {
			if !filter.matchComponent(component) {
				continue
			}

			// Create a unique key for component grouping (name + version + OS)
			// This matches the GROUP BY (component, version, operating_system) in the SQL view
			key := fmt.Sprintf("%s:%s:%s", component.GetName(), component.GetVersion(), image.GetScan().GetOperatingSystem())

			val := componentMap[key]
			if val == nil {
				// Generate component ID for this specific component in this image
				componentID := scancomponent.ComponentIDV2(component, image.GetId(), idx)

				// Initialize new component entry
				topCvss := component.GetTopCvss()
				riskScore := component.GetRiskScore()
				val = &imageComponentFlatResponse{
					Component:       component.GetName(),
					ComponentIDs:    []string{componentID},
					Version:         component.GetVersion(),
					TopCVSS:         &topCvss,
					RiskScore:       &riskScore,
					OperatingSystem: image.GetScan().GetOperatingSystem(),
				}
				componentMap[key] = val
			} else {
				// Aggregate data for the same component (same name+version+OS across different images)
				componentID := scancomponent.ComponentIDV2(component, image.GetId(), idx)

				// Add this component ID to the list (DISTINCT aggregation in SQL)
				val.ComponentIDs = append(val.ComponentIDs, componentID)

				// Take maximum CVSS and risk scores (MAX aggregation in SQL)
				if component.GetTopCvss() > val.GetTopCVSS() {
					topCvss := component.GetTopCvss()
					val.TopCVSS = &topCvss
				}
				if component.GetRiskScore() > val.GetRiskScore() {
					riskScore := component.GetRiskScore()
					val.RiskScore = &riskScore
				}
			}
		}
	}

	// Convert map to slice and sort component IDs for consistent results
	expected := make([]*imageComponentFlatResponse, 0, len(componentMap))
	for _, entry := range componentMap {
		// Sort component IDs for consistent results (matches the view implementation)
		sort.Strings(entry.ComponentIDs)
		expected = append(expected, entry)
	}

	// Apply sorting if specified, otherwise use default deterministic sorting
	if less != nil {
		sort.SliceStable(expected, less(expected))
	} else {
		// Default sorting by component name, then version, then OS for deterministic results
		sort.SliceStable(expected, func(i, j int) bool {
			if expected[i].Component != expected[j].Component {
				return expected[i].Component < expected[j].Component
			}
			if expected[i].Version != expected[j].Version {
				return expected[i].Version < expected[j].Version
			}
			return expected[i].OperatingSystem < expected[j].OperatingSystem
		})
	}

	// Convert to interface slice
	ret := make([]ComponentFlat, 0, len(expected))
	for _, entry := range expected {
		ret = append(ret, entry)
	}
	return ret
}

// compileExpectedWithSAC builds expected results with SAC filtering applied
func (s *ImageComponentFlatViewTestSuite) compileExpectedWithSAC(sacKey string, filter *filterImpl, less lessFunc) []ComponentFlat {
	// Get the SAC visibility map for this context
	sacTestCases := s.sacTestCases()
	visibilityMap, exists := sacTestCases[sacKey]
	s.Require().True(exists, "SAC test case %s should exist", sacKey)

	// Filter images based on SAC visibility
	var visibleImages []*storage.ImageV2
	for _, image := range s.testImages {
		if visible, exists := visibilityMap[image.GetId()]; exists && visible {
			visibleImages = append(visibleImages, image)
		}
	}

	// Use the existing compileExpected logic with the SAC-filtered images
	return s.compileExpected(visibleImages, filter, less)
}

func (s *ImageComponentFlatViewTestSuite) assertComponentResponsesAreEqual(t *testing.T, expected []ComponentFlat, actual []ComponentFlat, testOrder bool) {
	if !testOrder {
		// Use deterministic sorting for comparison when order doesn't matter
		sortFunc := func(i, j int, items []ComponentFlat) bool {
			if items[i].GetComponent() != items[j].GetComponent() {
				return items[i].GetComponent() < items[j].GetComponent()
			}
			if items[i].GetVersion() != items[j].GetVersion() {
				return items[i].GetVersion() < items[j].GetVersion()
			}
			return items[i].GetOperatingSystem() < items[j].GetOperatingSystem()
		}
		sort.SliceStable(expected, func(i, j int) bool {
			return sortFunc(i, j, expected)
		})
		sort.SliceStable(actual, func(i, j int) bool {
			return sortFunc(i, j, actual)
		})
	}

	assert.Equal(t, len(expected), len(actual))
	for i := range expected {
		assert.Equal(t, expected[i].GetComponent(), actual[i].GetComponent())
		assert.Equal(t, expected[i].GetVersion(), actual[i].GetVersion())
		assert.Equal(t, expected[i].GetOperatingSystem(), actual[i].GetOperatingSystem())
		assert.Equal(t, expected[i].GetTopCVSS(), actual[i].GetTopCVSS())
		assert.Equal(t, expected[i].GetRiskScore(), actual[i].GetRiskScore())
		assert.ElementsMatch(t, expected[i].GetComponentIDs(), actual[i].GetComponentIDs())
	}
}
