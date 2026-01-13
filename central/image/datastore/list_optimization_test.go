//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/image/datastore/keyfence"
	pgStoreV2 "github.com/stackrox/rox/central/image/datastore/store/v2/postgres"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestImageListOptimization(t *testing.T) {
	suite.Run(t, new(ImageListOptimizationTestSuite))
}

type ImageListOptimizationTestSuite struct {
	suite.Suite
	ctx       context.Context
	testDB    *pgtest.TestPostgres
	datastore DataStore
	mockRisk  *mockRisks.MockDataStore
}

func (s *ImageListOptimizationTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	s.testDB = pgtest.ForT(s.T())
}

func (s *ImageListOptimizationTestSuite) SetupTest() {
	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	dbStore := pgStoreV2.New(s.testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	s.datastore = NewWithPostgres(dbStore, s.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())
}

func (s *ImageListOptimizationTestSuite) TearDownSuite() {
	// pgtest handles cleanup automatically
}

func (s *ImageListOptimizationTestSuite) TearDownTest() {
	// Clean up images table after each test
	_, err := s.testDB.DB.Exec(s.ctx, "TRUNCATE images_v2 CASCADE")
	s.Require().NoError(err)
}

// TestOptimizedVsLegacyEquivalence verifies that optimized and legacy paths produce identical results
func (s *ImageListOptimizationTestSuite) TestOptimizedVsLegacyEquivalence() {
	// Create test images with various scan states
	images := []*storage.Image{
		s.createImageWithScan("sha1", "image1:v1", 10, 5, 2),      // Full scan data
		s.createImageWithScan("sha2", "image2:v2", 20, 10, 5),     // Different scan data
		s.createImageWithScan("sha3", "image3:v3", 0, 0, 0),       // Scanned but no CVEs
		s.createImageWithoutScan("sha4", "image4:v4"),             // No scan data (NULLs)
		s.createImageWithPartialScan("sha5", "image5:v5", 15, 0),  // Only components, no CVEs
	}

	// Insert test images
	for _, img := range images {
		s.Require().NoError(s.datastore.UpsertImage(s.ctx, img))
	}

	testCases := []struct {
		name  string
		query *v1.Query
	}{
		{
			name:  "empty query (all images)",
			query: pkgSearch.EmptyQuery(),
		},
		{
			name: "with pagination",
			query: pkgSearch.NewQueryBuilder().
				WithPagination(pkgSearch.NewPagination().Limit(3)).
				ProtoQuery(),
		},
		{
			name: "with sorting by name",
			query: pkgSearch.NewQueryBuilder().
				WithPagination(pkgSearch.NewPagination().
					AddSortOption(pkgSearch.NewSortOption(pkgSearch.ImageName))).
				ProtoQuery(),
		},
		{
			name: "filtered by image name",
			query: pkgSearch.NewQueryBuilder().
				AddStrings(pkgSearch.ImageName, "image1").
				ProtoQuery(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Get results with feature flag disabled (legacy path)
			s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "false")
			// Feature flag controlled via s.T().Setenv above
			legacyResults, err := s.datastore.SearchListImages(s.ctx, tc.query)
			s.Require().NoError(err)

			// Get results with feature flag enabled (optimized path)
			s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
			// Feature flag controlled via s.T().Setenv above
			optimizedResults, err := s.datastore.SearchListImages(s.ctx, tc.query)
			s.Require().NoError(err)

			// Verify same number of results
			s.Require().Equal(len(legacyResults), len(optimizedResults),
				"Result count mismatch between legacy and optimized paths")

			// Compare results field-by-field
			for i := range legacyResults {
				s.assertListImageEqual(legacyResults[i], optimizedResults[i])
			}
		})
	}
}

// TestNullHandling verifies proper handling of NULL scan stats
func (s *ImageListOptimizationTestSuite) TestNullHandling() {
	s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
	// Feature flag controlled via s.T().Setenv above

	testCases := []struct {
		name                 string
		image                *storage.Image
		expectComponentsSet  bool
		expectCvesSet        bool
		expectFixableSet     bool
		expectedComponents   int32
		expectedCves         int32
		expectedFixable      int32
	}{
		{
			name:                "image with full scan data",
			image:               s.createImageWithScan("sha-full", "full:v1", 10, 5, 2),
			expectComponentsSet: true,
			expectCvesSet:       true,
			expectFixableSet:    true,
			expectedComponents:  10,
			expectedCves:        5,
			expectedFixable:     2,
		},
		{
			name:                "image without scan (all NULLs)",
			image:               s.createImageWithoutScan("sha-null", "null:v1"),
			expectComponentsSet: false,
			expectCvesSet:       false,
			expectFixableSet:    false,
		},
		{
			name:                "image with zero CVEs",
			image:               s.createImageWithScan("sha-zero", "zero:v1", 5, 0, 0),
			expectComponentsSet: true,
			expectCvesSet:       true,
			expectFixableSet:    true,
			expectedComponents:  5,
			expectedCves:        0,
			expectedFixable:     0,
		},
		{
			name:                "image with partial scan",
			image:               s.createImageWithPartialScan("sha-partial", "partial:v1", 8, 3),
			expectComponentsSet: true,
			expectCvesSet:       true,
			expectFixableSet:    false,
			expectedComponents:  8,
			expectedCves:        3,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Insert image
			s.Require().NoError(s.datastore.UpsertImage(s.ctx, tc.image))

			// Query for the image
			query := pkgSearch.NewQueryBuilder().
				AddStrings(pkgSearch.ImageSHA, tc.image.GetId()).
				ProtoQuery()
			results, err := s.datastore.SearchListImages(s.ctx, query)
			s.Require().NoError(err)
			s.Require().Len(results, 1)

			listImg := results[0]

			// Verify oneof fields
			if tc.expectComponentsSet {
				s.Require().NotNil(listImg.GetSetComponents(),
					"SetComponents should be set")
				s.Equal(tc.expectedComponents, listImg.GetComponents())
			} else {
				s.Nil(listImg.GetSetComponents(),
					"SetComponents should be nil for NULL database value")
			}

			if tc.expectCvesSet {
				s.Require().NotNil(listImg.GetSetCves(),
					"SetCves should be set")
				s.Equal(tc.expectedCves, listImg.GetCves())
			} else {
				s.Nil(listImg.GetSetCves(),
					"SetCves should be nil for NULL database value")
			}

			if tc.expectFixableSet {
				s.Require().NotNil(listImg.GetSetFixable(),
					"SetFixable should be set")
				s.Equal(tc.expectedFixable, listImg.GetFixableCves())
			} else {
				s.Nil(listImg.GetSetFixable(),
					"SetFixable should be nil for NULL database value")
			}

			// Clean up
			_, err = s.testDB.DB.Exec(s.ctx, "DELETE FROM images_v2 WHERE id = $1", tc.image.GetId())
			s.Require().NoError(err)
		})
	}
}

// TestFeatureFlagToggle verifies feature flag controls routing
func (s *ImageListOptimizationTestSuite) TestFeatureFlagToggle() {
	// Insert a test image
	img := s.createImageWithScan("sha-flag", "flag:v1", 10, 5, 2)
	s.Require().NoError(s.datastore.UpsertImage(s.ctx, img))

	query := pkgSearch.EmptyQuery()

	// Test with flag disabled
	s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "false")
	// Feature flag controlled via s.T().Setenv above
	s.False(features.ImageListOptimization.Enabled(), "Feature flag should be disabled")

	results, err := s.datastore.SearchListImages(s.ctx, query)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	disabledResult := results[0]

	// Test with flag enabled
	s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
	// Feature flag controlled via s.T().Setenv above
	s.True(features.ImageListOptimization.Enabled(), "Feature flag should be enabled")

	results, err = s.datastore.SearchListImages(s.ctx, query)
	s.Require().NoError(err)
	s.Require().Len(results, 1)
	enabledResult := results[0]

	// Results should be identical
	s.assertListImageEqual(disabledResult, enabledResult)
}

// TestTimestampConversion verifies timestamp handling
func (s *ImageListOptimizationTestSuite) TestTimestampConversion() {
	s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
	// Feature flag controlled via s.T().Setenv above

	now := time.Now().Truncate(time.Microsecond) // Postgres precision
	createdTime := protocompat.ConvertTimeToTimestampOrNil(&now)

	img := fixtures.GetImage()
	img.Id = "sha-timestamp"
	img.Name = &storage.ImageName{FullName: "timestamp:v1"}
	img.Metadata = &storage.ImageMetadata{
		V1: &storage.V1Metadata{
			Created: createdTime,
		},
	}
	img.LastUpdated = createdTime
	img.SetComponents = &storage.Image_Components{Components: 5}
	img.SetCves = &storage.Image_Cves{Cves: 2}
	img.SetFixable = &storage.Image_FixableCves{FixableCves: 1}

	s.Require().NoError(s.datastore.UpsertImage(s.ctx, img))

	query := pkgSearch.NewQueryBuilder().
		AddStrings(pkgSearch.ImageSHA, "sha-timestamp").
		ProtoQuery()
	results, err := s.datastore.SearchListImages(s.ctx, query)
	s.Require().NoError(err)
	s.Require().Len(results, 1)

	listImg := results[0]
	s.NotNil(listImg.GetCreated())
	s.NotNil(listImg.GetLastUpdated())

	// Compare timestamps (allowing for minor precision differences)
	s.True(protocompat.CompareTimestamps(createdTime, listImg.GetCreated()) == 0,
		"Created timestamp mismatch")
	s.True(protocompat.CompareTimestamps(createdTime, listImg.GetLastUpdated()) == 0,
		"LastUpdated timestamp mismatch")
}

// Helper functions

func (s *ImageListOptimizationTestSuite) createImageWithScan(sha, name string, components, cves, fixable int32) *storage.Image {
	img := fixtures.GetImage()
	img.Id = sha
	img.Name = &storage.ImageName{FullName: name}
	img.SetComponents = &storage.Image_Components{Components: components}
	img.SetCves = &storage.Image_Cves{Cves: cves}
	img.SetFixable = &storage.Image_FixableCves{FixableCves: fixable}
	return img
}

func (s *ImageListOptimizationTestSuite) createImageWithoutScan(sha, name string) *storage.Image {
	img := fixtures.GetImage()
	img.Id = sha
	img.Name = &storage.ImageName{FullName: name}
	// Clear scan data to simulate unscanned image
	img.SetComponents = nil
	img.SetCves = nil
	img.SetFixable = nil
	img.Scan = nil
	return img
}

func (s *ImageListOptimizationTestSuite) createImageWithPartialScan(sha, name string, components, cves int32) *storage.Image {
	img := fixtures.GetImage()
	img.Id = sha
	img.Name = &storage.ImageName{FullName: name}
	img.SetComponents = &storage.Image_Components{Components: components}
	img.SetCves = &storage.Image_Cves{Cves: cves}
	img.SetFixable = nil // NULL fixable count
	return img
}

func (s *ImageListOptimizationTestSuite) assertListImageEqual(expected, actual *storage.ListImage) {
	s.Equal(expected.GetId(), actual.GetId(), "ID mismatch")
	s.Equal(expected.GetName(), actual.GetName(), "Name mismatch")

	// Compare oneof fields
	if expected.GetSetComponents() != nil {
		s.Require().NotNil(actual.GetSetComponents(), "SetComponents should be set")
		s.Equal(expected.GetComponents(), actual.GetComponents(), "Components mismatch")
	} else {
		s.Nil(actual.GetSetComponents(), "SetComponents should be nil")
	}

	if expected.GetSetCves() != nil {
		s.Require().NotNil(actual.GetSetCves(), "SetCves should be set")
		s.Equal(expected.GetCves(), actual.GetCves(), "CVEs mismatch")
	} else {
		s.Nil(actual.GetSetCves(), "SetCves should be nil")
	}

	if expected.GetSetFixable() != nil {
		s.Require().NotNil(actual.GetSetFixable(), "SetFixable should be set")
		s.Equal(expected.GetFixableCves(), actual.GetFixableCves(), "Fixable CVEs mismatch")
	} else {
		s.Nil(actual.GetSetFixable(), "SetFixable should be nil")
	}

	// Compare timestamps
	if expected.GetCreated() != nil {
		s.Require().NotNil(actual.GetCreated(), "Created timestamp should be set")
		s.True(protocompat.CompareTimestamps(expected.GetCreated(), actual.GetCreated()) == 0,
			"Created timestamp mismatch")
	} else {
		s.Nil(actual.GetCreated(), "Created timestamp should be nil")
	}

	if expected.GetLastUpdated() != nil {
		s.Require().NotNil(actual.GetLastUpdated(), "LastUpdated timestamp should be set")
		s.True(protocompat.CompareTimestamps(expected.GetLastUpdated(), actual.GetLastUpdated()) == 0,
			"LastUpdated timestamp mismatch")
	} else {
		s.Nil(actual.GetLastUpdated(), "LastUpdated timestamp should be nil")
	}

	// Priority is set by ranker, so both should have same priority
	// (either both set or both unset)
	s.Equal(expected.GetPriority(), actual.GetPriority(), "Priority mismatch")
}

// TestLargeResultSet verifies optimized query handles large result sets
func (s *ImageListOptimizationTestSuite) TestLargeResultSet() {
	s.T().Setenv("ROX_IMAGE_LIST_OPTIMIZATION", "true")
	// Feature flag controlled via s.T().Setenv above

	// Insert 100 test images
	imageCount := 100
	imageIDs := set.NewStringSet()
	for i := 0; i < imageCount; i++ {
		sha := fmt.Sprintf("sha-large-%d", i)
		name := fmt.Sprintf("large/image-%d:v1", i)
		img := s.createImageWithScan(
			sha,
			name,
			int32(i%20),
			int32(i%10),
			int32(i%5),
		)
		imageIDs.Add(img.GetId())
		s.Require().NoError(s.datastore.UpsertImage(s.ctx, img))
	}

	// Query all images
	results, err := s.datastore.SearchListImages(s.ctx, pkgSearch.EmptyQuery())
	s.Require().NoError(err)
	s.Equal(imageCount, len(results), "Should return all images")

	// Verify all images are present
	resultIDs := set.NewStringSet()
	for _, img := range results {
		resultIDs.Add(img.GetId())
	}
	s.True(imageIDs.Equal(resultIDs), "All images should be returned")
}
