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
	// Clean database before initializing datastore to ensure rankers start fresh
	// Use context.Background() to match what initializeRankers uses
	cleanCtx := context.Background()
	_, err := s.testDB.DB.Exec(cleanCtx, "TRUNCATE images_v2 CASCADE")
	s.Require().NoError(err)

	s.mockRisk = mockRisks.NewMockDataStore(gomock.NewController(s.T()))
	dbStore := pgStoreV2.New(s.testDB.DB, false, keyfence.ImageKeyFenceSingleton())
	s.datastore = NewWithPostgres(dbStore, s.mockRisk, ranking.ImageRanker(), ranking.ComponentRanker())
}

func (s *ImageListOptimizationTestSuite) TearDownSuite() {
	// pgtest handles cleanup automatically
}

func (s *ImageListOptimizationTestSuite) TearDownTest() {
	// Clean up images table after each test method to ensure isolation
	_, err := s.testDB.DB.Exec(s.ctx, "TRUNCATE images_v2 CASCADE")
	s.Require().NoError(err)
}

// TestSearchListImagesWithVariousQueries verifies optimized queries work with different patterns
func (s *ImageListOptimizationTestSuite) TestSearchListImagesWithVariousQueries() {
	// Create test images with various scan states
	// Note: FillScanStats calculates CVEs as (components * 5) unique CVEs
	images := []*storage.Image{
		s.createImageWithScan("sha1", "image1:v1", 10, 50, 50),    // 10 components, 50 CVEs, 50 fixable
		s.createImageWithScan("sha2", "image2:v2", 20, 100, 100),  // 20 components, 100 CVEs, 100 fixable
		s.createImageWithScan("sha3", "image3:v3", 0, 0, 0),       // 0 components, 0 CVEs, 0 fixable
		s.createImageWithoutScan("sha4", "image4:v4"),             // No scan data (NULLs)
		s.createImageWithPartialScan("sha5", "image5:v5", 15, 75), // 15 components, 75 CVEs, no fixable
	}

	// Insert test images
	for _, img := range images {
		s.Require().NoError(s.datastore.UpsertImage(s.ctx, img))
	}

	testCases := []struct {
		name          string
		query         *v1.Query
		expectedCount int
	}{
		{
			name:          "empty query (all images)",
			query:         pkgSearch.EmptyQuery(),
			expectedCount: 5,
		},
		{
			name: "with pagination",
			query: pkgSearch.NewQueryBuilder().
				WithPagination(pkgSearch.NewPagination().Limit(3)).
				ProtoQuery(),
			expectedCount: 3,
		},
		{
			name: "with sorting by name",
			query: pkgSearch.NewQueryBuilder().
				WithPagination(pkgSearch.NewPagination().
					AddSortOption(pkgSearch.NewSortOption(pkgSearch.ImageName))).
				ProtoQuery(),
			expectedCount: 5,
		},
		{
			name: "filtered by image name",
			query: pkgSearch.NewQueryBuilder().
				AddStrings(pkgSearch.ImageName, "image1").
				ProtoQuery(),
			expectedCount: 1,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			results, err := s.datastore.SearchListImages(s.ctx, tc.query)
			s.Require().NoError(err)
			s.Require().Len(results, tc.expectedCount, "Unexpected result count")

			// Verify all results have required fields
			for _, img := range results {
				s.NotEmpty(img.GetId(), "ID should be set")
				s.NotEmpty(img.GetName(), "Name should be set")
			}
		})
	}
}

// TestNullHandling verifies proper handling of NULL scan stats
func (s *ImageListOptimizationTestSuite) TestNullHandling() {
	testCases := []struct {
		name                string
		image               *storage.Image
		expectComponentsSet bool
		expectCvesSet       bool
		expectFixableSet    bool
		expectedComponents  int32
		expectedCves        int32
		expectedFixable     int32
	}{
		{
			name: "image with full scan data",
			// GetImageWithUniqueComponents(10) creates 10 components, each with 5 unique CVEs = 50 CVEs total
			// FillScanStats will calculate: Components=10, CVEs=50, FixableCVEs=50
			image:               s.createImageWithScan("sha-full", "full:v1", 10, 50, 50),
			expectComponentsSet: true,
			expectCvesSet:       true,
			expectFixableSet:    true,
			expectedComponents:  10,
			expectedCves:        50,
			expectedFixable:     50,
		},
		{
			name:                "image without scan (all NULLs)",
			image:               s.createImageWithoutScan("sha-null", "null:v1"),
			expectComponentsSet: false,
			expectCvesSet:       false,
			expectFixableSet:    false,
		},
		{
			name: "image with zero CVEs",
			// GetImageWithUniqueComponents(5) creates 5 components with 5 unique CVEs each = 25 total
			// FillScanStats will calculate: Components=5, CVEs=25, FixableCVEs=25
			image:               s.createImageWithScan("sha-zero", "zero:v1", 5, 25, 25),
			expectComponentsSet: true,
			expectCvesSet:       true,
			expectFixableSet:    true,
			expectedComponents:  5,
			expectedCves:        25,
			expectedFixable:     25,
		},
		{
			name: "image with partial scan",
			// GetImageWithUniqueComponents(8) creates 8 components with 5 unique CVEs each = 40 total
			// Fixable count should NOT be set because we remove FixedBy from all vulns
			image:               s.createImageWithPartialScan("sha-partial", "partial:v1", 8, 40),
			expectComponentsSet: true,
			expectCvesSet:       true,
			expectFixableSet:    false,
			expectedComponents:  8,
			expectedCves:        40,
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

// TestTimestampConversion verifies timestamp handling
func (s *ImageListOptimizationTestSuite) TestTimestampConversion() {
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
	// Create image with specific scan component count to match expected values
	// Note: FillScanStats in UpsertImage will recalculate these from actual Scan data
	img := fixtures.GetImageWithUniqueComponents(int(components))
	img.Id = sha
	img.Name = &storage.ImageName{FullName: name}

	// Set the oneof fields - these will be recalculated by FillScanStats based on actual Scan data
	// but we set them here so the test data expectations match
	img.SetComponents = &storage.Image_Components{Components: components}
	img.SetCves = &storage.Image_Cves{Cves: cves}
	img.SetFixable = &storage.Image_FixableCves{FixableCves: fixable}
	return img
}

func (s *ImageListOptimizationTestSuite) createImageWithoutScan(sha, name string) *storage.Image {
	// Create minimal image without any scan data
	now := time.Now()
	img := &storage.Image{
		Id:   sha,
		Name: &storage.ImageName{FullName: name},
		Metadata: &storage.ImageMetadata{
			V1: &storage.V1Metadata{
				Created: protocompat.ConvertTimeToTimestampOrNil(&now),
			},
		},
		LastUpdated: protocompat.ConvertTimeToTimestampOrNil(&now),
		// No Scan data - this will result in NULL database values
		Scan: nil,
	}
	return img
}

func (s *ImageListOptimizationTestSuite) createImageWithPartialScan(sha, name string, components, cves int32) *storage.Image {
	// Create image with specific scan component count
	img := fixtures.GetImageWithUniqueComponents(int(components))
	img.Id = sha
	img.Name = &storage.ImageName{FullName: name}

	// Set components and CVEs, but leave fixable as nil
	img.SetComponents = &storage.Image_Components{Components: components}
	img.SetCves = &storage.Image_Cves{Cves: cves}
	img.SetFixable = nil // This should remain NULL in database

	// Remove FixedBy from all vulns to ensure fixable count is not calculated
	if img.Scan != nil {
		for _, component := range img.Scan.Components {
			for _, vuln := range component.Vulns {
				vuln.SetFixedBy = nil
			}
		}
	}
	return img
}

// TestLargeResultSet verifies optimized query handles large result sets
func (s *ImageListOptimizationTestSuite) TestLargeResultSet() {
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
