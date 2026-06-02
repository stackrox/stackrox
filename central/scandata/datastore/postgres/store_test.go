//go:build sql_integration

package postgres

import (
	"context"
	"testing"
	"time"

	imageV2DS "github.com/stackrox/rox/central/imagev2/datastore"
	imageV2Store "github.com/stackrox/rox/central/imagev2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDatastoreMocks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/central/scandata/datastore"
	"github.com/stackrox/rox/central/scandata/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ScanDataStoreTestSuite struct {
	suite.Suite
	ctx        context.Context
	store      datastore.DataStore
	imageStore imageV2DS.DataStore
}

func (s *ScanDataStoreTestSuite) SetupSuite() {
	s.ctx = sac.WithAllAccess(context.Background())
	testingDB := pgtest.ForT(s.T())
	s.store = New(testingDB.DB)

	// Create image store to satisfy foreign key constraints
	ctrl := gomock.NewController(s.T())
	mockRiskDatastore := riskDatastoreMocks.NewMockDataStore(ctrl)
	mockRiskDatastore.EXPECT().RemoveRisk(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	s.imageStore = imageV2DS.NewWithPostgres(
		imageV2Store.New(testingDB.DB, true, concurrency.NewKeyFence()),
		mockRiskDatastore,
		ranking.ImageRanker(),
		ranking.ComponentRanker(),
	)
}

// createTestImage creates a minimal image record to satisfy foreign key constraints
func (s *ScanDataStoreTestSuite) createTestImage(imageID string) {
	img := &storage.ImageV2{
		Id:     imageID,
		Digest: "sha256:test",
		Name: &storage.ImageName{
			FullName: "test-image:latest",
		},
	}
	err := s.imageStore.UpsertImage(s.ctx, img)
	require.NoError(s.T(), err)
}

// deleteTestImage removes the test image
func (s *ScanDataStoreTestSuite) deleteTestImage(imageID string) {
	err := s.imageStore.DeleteImages(s.ctx, imageID)
	require.NoError(s.T(), err)
}

func TestScanDataStore(t *testing.T) {
	suite.Run(t, new(ScanDataStoreTestSuite))
}

func (s *ScanDataStoreTestSuite) TestUpsertAndGetScanData() {
	ctx := s.ctx
	imageID := "test-image-1"
	scanID := "scan-1"

	// Create image first to satisfy foreign key constraint
	s.createTestImage(imageID)
	defer s.deleteTestImage(imageID)

	now := timestamppb.Now()

	// Create scan data with 1 scan, 2 components, 3 findings
	scanData := &types.ScanData{
		Scan: &storage.ImageScanV2{
			Id:             scanID,
			ImageId:        imageID,
			ScanTime:       now,
			ScannerVersion: "scanner-v4.0",
			BundleVersion:  "bundle-2024.01",
		},
		Components: []*storage.ScanComponent{
			{
				Id:              "comp-1",
				ScanId:          scanID,
				ImageId:         imageID,
				Name:            "openssl",
				Version:         "1.1.1",
				Source:          storage.SourceType_OS,
				Location:        "/usr/lib",
				HasLayerIndex:   &storage.ScanComponent_LayerIndex{LayerIndex: 0},
				LayerType:       storage.LayerType_APPLICATION,
				FixedBy:         "1.1.2",
				OperatingSystem: "alpine:3.18",
			},
			{
				Id:              "comp-2",
				ScanId:          scanID,
				ImageId:         imageID,
				Name:            "curl",
				Version:         "7.88.0",
				Source:          storage.SourceType_OS,
				Location:        "/usr/bin",
				HasLayerIndex:   &storage.ScanComponent_LayerIndex{LayerIndex: 1},
				LayerType:       storage.LayerType_APPLICATION,
				FixedBy:         "",
				OperatingSystem: "alpine:3.18",
			},
		},
		Findings: []*storage.ScanFinding{
			{
				Id:                    "adv-1#comp-1",
				AdvisoryId:            "adv-1",
				CveName:               "CVE-2024-1111",
				ComponentId:           "comp-1",
				ScanId:                scanID,
				ImageId:               imageID,
				Severity:              storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				Cvss:                  9.8,
				CvssVersion:           storage.CvssScoreVersion_V3,
				NvdCvss:               9.8,
				NvdCvssVersion:        storage.CvssScoreVersion_V3,
				EpssProbability:       0.95,
				EpssPercentile:        0.99,
				IsFixable:             true,
				FixedBy:               "1.1.2",
				FixedDate:             timestamppb.New(time.Now().Add(-30 * 24 * time.Hour)),
				Description:           "Critical vulnerability in openssl",
				PublishedDate:         timestamppb.New(time.Now().Add(-60 * 24 * time.Hour)),
				DataSource:            "NVD",
				SourceName:            "nvd",
				Links:                 []string{"https://nvd.nist.gov/vuln/detail/CVE-2024-1111"},
				State:                 storage.VulnerabilityState_OBSERVED,
				FirstImageOccurrence:  now,
				FirstSystemOccurrence: now,
			},
			{
				Id:                    "adv-2#comp-1",
				AdvisoryId:            "adv-2",
				CveName:               "CVE-2024-2222",
				ComponentId:           "comp-1",
				ScanId:                scanID,
				ImageId:               imageID,
				Severity:              storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				Cvss:                  5.3,
				CvssVersion:           storage.CvssScoreVersion_V3,
				IsFixable:             false,
				Description:           "Moderate vulnerability in openssl",
				PublishedDate:         timestamppb.New(time.Now().Add(-90 * 24 * time.Hour)),
				DataSource:            "Alpine",
				SourceName:            "alpine",
				State:                 storage.VulnerabilityState_OBSERVED,
				FirstImageOccurrence:  now,
				FirstSystemOccurrence: now,
			},
			{
				Id:                    "adv-3#comp-2",
				AdvisoryId:            "adv-3",
				CveName:               "CVE-2024-3333",
				ComponentId:           "comp-2",
				ScanId:                scanID,
				ImageId:               imageID,
				Severity:              storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				Cvss:                  3.1,
				CvssVersion:           storage.CvssScoreVersion_V3,
				IsFixable:             false,
				Description:           "Low severity issue in curl",
				PublishedDate:         timestamppb.New(time.Now().Add(-45 * 24 * time.Hour)),
				DataSource:            "NVD",
				SourceName:            "nvd",
				State:                 storage.VulnerabilityState_OBSERVED,
				FirstImageOccurrence:  now,
				FirstSystemOccurrence: now,
			},
		},
	}

	// Upsert the scan data
	err := s.store.UpsertScanData(ctx, scanData)
	require.NoError(s.T(), err)

	// Get the scan data back
	retrieved, err := s.store.GetScanDataByImageID(ctx, imageID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), retrieved)

	// Verify scan
	assert.Equal(s.T(), scanID, retrieved.Scan.GetId())
	assert.Equal(s.T(), imageID, retrieved.Scan.GetImageId())
	assert.Equal(s.T(), "scanner-v4.0", retrieved.Scan.GetScannerVersion())

	// Verify component count
	require.Len(s.T(), retrieved.Components, 2)

	// Verify findings count
	require.Len(s.T(), retrieved.Findings, 3)

	// Verify finding details
	findingCVEs := make(map[string]bool)
	for _, f := range retrieved.Findings {
		findingCVEs[f.GetCveName()] = true
		assert.Equal(s.T(), imageID, f.GetImageId())
		assert.Equal(s.T(), scanID, f.GetScanId())
	}
	assert.True(s.T(), findingCVEs["CVE-2024-1111"])
	assert.True(s.T(), findingCVEs["CVE-2024-2222"])
	assert.True(s.T(), findingCVEs["CVE-2024-3333"])

	// Cleanup
	err = s.store.DeleteByImageID(ctx, imageID)
	require.NoError(s.T(), err)
}

func (s *ScanDataStoreTestSuite) TestUpsertReplacesOldData() {
	ctx := s.ctx
	imageID := "test-image-2"

	// Create image first to satisfy foreign key constraint
	s.createTestImage(imageID)
	defer s.deleteTestImage(imageID)

	// Insert first scan data
	firstScan := &types.ScanData{
		Scan: &storage.ImageScanV2{
			Id:             "scan-old",
			ImageId:        imageID,
			ScanTime:       timestamppb.New(time.Now().Add(-24 * time.Hour)),
			ScannerVersion: "scanner-v3.0",
			BundleVersion:  "bundle-2023.12",
		},
		Components: []*storage.ScanComponent{
			{
				Id:      "old-comp-1",
				ScanId:  "scan-old",
				ImageId: imageID,
				Name:    "old-package",
				Version: "1.0.0",
			},
		},
		Findings: []*storage.ScanFinding{
			{
				Id:          "old-finding-1",
				AdvisoryId:  "old-adv-1",
				CveName:     "CVE-2023-9999",
				ComponentId: "old-comp-1",
				ScanId:      "scan-old",
				ImageId:     imageID,
				Severity:    storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				State:       storage.VulnerabilityState_OBSERVED,
			},
		},
	}

	err := s.store.UpsertScanData(ctx, firstScan)
	require.NoError(s.T(), err)

	// Insert new scan data for same image
	newScan := &types.ScanData{
		Scan: &storage.ImageScanV2{
			Id:             "scan-new",
			ImageId:        imageID,
			ScanTime:       timestamppb.Now(),
			ScannerVersion: "scanner-v4.0",
			BundleVersion:  "bundle-2024.01",
		},
		Components: []*storage.ScanComponent{
			{
				Id:      "new-comp-1",
				ScanId:  "scan-new",
				ImageId: imageID,
				Name:    "new-package",
				Version: "2.0.0",
			},
		},
		Findings: []*storage.ScanFinding{
			{
				Id:          "new-finding-1",
				AdvisoryId:  "new-adv-1",
				CveName:     "CVE-2024-8888",
				ComponentId: "new-comp-1",
				ScanId:      "scan-new",
				ImageId:     imageID,
				Severity:    storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
				State:       storage.VulnerabilityState_OBSERVED,
			},
		},
	}

	err = s.store.UpsertScanData(ctx, newScan)
	require.NoError(s.T(), err)

	// Retrieve and verify only new data exists
	retrieved, err := s.store.GetScanDataByImageID(ctx, imageID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), retrieved)

	// Verify scan is the new one
	assert.Equal(s.T(), "scan-new", retrieved.Scan.GetId())
	assert.Equal(s.T(), "scanner-v4.0", retrieved.Scan.GetScannerVersion())

	// Verify old component is gone
	require.Len(s.T(), retrieved.Components, 1)
	assert.Equal(s.T(), "new-comp-1", retrieved.Components[0].GetId())
	assert.Equal(s.T(), "new-package", retrieved.Components[0].GetName())

	// Verify old finding is gone
	require.Len(s.T(), retrieved.Findings, 1)
	assert.Equal(s.T(), "CVE-2024-8888", retrieved.Findings[0].GetCveName())

	// Cleanup
	err = s.store.DeleteByImageID(ctx, imageID)
	require.NoError(s.T(), err)
}

func (s *ScanDataStoreTestSuite) TestListCVEs() {
	ctx := s.ctx

	// Create test images first
	imageIDs := []string{"img-1", "img-2", "img-3", "img-4"}
	for _, id := range imageIDs {
		s.createTestImage(id)
	}
	defer func() {
		for _, id := range imageIDs {
			s.deleteTestImage(id)
		}
	}()

	// Insert findings for multiple images with different CVEs
	images := []struct {
		imageID  string
		scanID   string
		cve      string
		severity storage.VulnerabilitySeverity
		cvss     float32
		fixable  bool
	}{
		{"img-1", "scan-img-1", "CVE-2024-AAAA", storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, 9.8, true},
		{"img-2", "scan-img-2", "CVE-2024-AAAA", storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, 6.5, false}, // Same CVE, different severity
		{"img-3", "scan-img-3", "CVE-2024-BBBB", storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY, 5.3, false},
		{"img-4", "scan-img-4", "CVE-2024-CCCC", storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY, 2.1, false},
	}

	now := timestamppb.Now()

	for _, img := range images {
		scanData := &types.ScanData{
			Scan: &storage.ImageScanV2{
				Id:       img.scanID,
				ImageId:  img.imageID,
				ScanTime: now,
			},
			Components: []*storage.ScanComponent{
				{
					Id:      "comp-" + img.imageID,
					ScanId:  img.scanID,
					ImageId: img.imageID,
					Name:    "test-package",
					Version: "1.0",
				},
			},
			Findings: []*storage.ScanFinding{
				{
					Id:                    "finding-" + img.imageID,
					AdvisoryId:            "adv-" + img.imageID,
					CveName:               img.cve,
					ComponentId:           "comp-" + img.imageID,
					ScanId:                img.scanID,
					ImageId:               img.imageID,
					Severity:              img.severity,
					Cvss:                  img.cvss,
					IsFixable:             img.fixable,
					State:                 storage.VulnerabilityState_OBSERVED,
					FirstSystemOccurrence: now,
				},
			},
		}
		err := s.store.UpsertScanData(ctx, scanData)
		require.NoError(s.T(), err)
	}

	// List CVEs
	rows, total, err := s.store.ListCVEs(ctx, 100, 0, "severity", "desc")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), 3, total) // 3 distinct CVEs

	// Verify aggregation for CVE-2024-AAAA
	var aaaaRow *types.CVEListRow
	for _, row := range rows {
		if row.CVEName == "CVE-2024-AAAA" {
			aaaaRow = row
			break
		}
	}
	require.NotNil(s.T(), aaaaRow, "CVE-2024-AAAA should be in results")

	// MAX severity should be CRITICAL (from img-1)
	assert.Equal(s.T(), int32(storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY), aaaaRow.Severity)

	// MAX cvss should be 9.8 (from img-1)
	assert.Equal(s.T(), float32(9.8), aaaaRow.CVSS)

	// COUNT DISTINCT images should be 2 (img-1, img-2)
	assert.Equal(s.T(), 2, aaaaRow.ImageCount)

	// BOOL_OR fixable should be true (img-1 is fixable)
	assert.True(s.T(), aaaaRow.Fixable)

	// Cleanup
	for _, img := range images {
		err := s.store.DeleteByImageID(ctx, img.imageID)
		require.NoError(s.T(), err)
	}
}

func (s *ScanDataStoreTestSuite) TestGetFindingsByCVE() {
	ctx := s.ctx

	// Create image first to satisfy foreign key constraint
	imageID := "img-test-cve"
	s.createTestImage(imageID)
	defer s.deleteTestImage(imageID)

	// Insert findings with different CVEs
	scanData := &types.ScanData{
		Scan: &storage.ImageScanV2{
			Id:       "scan-test-cve",
			ImageId:  "img-test-cve",
			ScanTime: timestamppb.Now(),
		},
		Components: []*storage.ScanComponent{
			{
				Id:      "comp-cve-1",
				ScanId:  "scan-test-cve",
				ImageId: "img-test-cve",
				Name:    "pkg1",
				Version: "1.0",
			},
			{
				Id:      "comp-cve-2",
				ScanId:  "scan-test-cve",
				ImageId: "img-test-cve",
				Name:    "pkg2",
				Version: "2.0",
			},
		},
		Findings: []*storage.ScanFinding{
			{
				Id:          "finding-target",
				AdvisoryId:  "adv-target",
				CveName:     "CVE-2024-1234",
				ComponentId: "comp-cve-1",
				ScanId:      "scan-test-cve",
				ImageId:     "img-test-cve",
				Severity:    storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				State:       storage.VulnerabilityState_OBSERVED,
			},
			{
				Id:          "finding-other",
				AdvisoryId:  "adv-other",
				CveName:     "CVE-2024-9999",
				ComponentId: "comp-cve-2",
				ScanId:      "scan-test-cve",
				ImageId:     "img-test-cve",
				Severity:    storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
				State:       storage.VulnerabilityState_OBSERVED,
			},
		},
	}

	err := s.store.UpsertScanData(ctx, scanData)
	require.NoError(s.T(), err)

	// Get findings for CVE-2024-1234
	findings, err := s.store.GetFindingsByCVE(ctx, "CVE-2024-1234")
	require.NoError(s.T(), err)

	// Should only return the one finding
	require.Len(s.T(), findings, 1)
	assert.Equal(s.T(), "CVE-2024-1234", findings[0].GetCveName())
	assert.Equal(s.T(), "finding-target", findings[0].GetId())

	// Verify other CVE not returned
	otherFindings, err := s.store.GetFindingsByCVE(ctx, "CVE-2024-9999")
	require.NoError(s.T(), err)
	require.Len(s.T(), otherFindings, 1)
	assert.Equal(s.T(), "CVE-2024-9999", otherFindings[0].GetCveName())

	// Cleanup
	err = s.store.DeleteByImageID(ctx, "img-test-cve")
	require.NoError(s.T(), err)
}

func (s *ScanDataStoreTestSuite) TestDeleteByImageID() {
	ctx := s.ctx
	imageID := "test-image-delete"

	// Create image first to satisfy foreign key constraint
	s.createTestImage(imageID)
	defer s.deleteTestImage(imageID)

	// Insert scan data
	scanData := &types.ScanData{
		Scan: &storage.ImageScanV2{
			Id:       "scan-delete",
			ImageId:  imageID,
			ScanTime: timestamppb.Now(),
		},
		Components: []*storage.ScanComponent{
			{
				Id:      "comp-delete",
				ScanId:  "scan-delete",
				ImageId: imageID,
				Name:    "pkg",
				Version: "1.0",
			},
		},
		Findings: []*storage.ScanFinding{
			{
				Id:          "finding-delete",
				AdvisoryId:  "adv-delete",
				CveName:     "CVE-2024-DELETE",
				ComponentId: "comp-delete",
				ScanId:      "scan-delete",
				ImageId:     imageID,
				Severity:    storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
				State:       storage.VulnerabilityState_OBSERVED,
			},
		},
	}

	err := s.store.UpsertScanData(ctx, scanData)
	require.NoError(s.T(), err)

	// Verify data exists
	retrieved, err := s.store.GetScanDataByImageID(ctx, imageID)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), retrieved)
	assert.Len(s.T(), retrieved.Components, 1)
	assert.Len(s.T(), retrieved.Findings, 1)

	// Delete by image ID
	err = s.store.DeleteByImageID(ctx, imageID)
	require.NoError(s.T(), err)

	// Verify data is gone
	retrieved, err = s.store.GetScanDataByImageID(ctx, imageID)
	require.NoError(s.T(), err)
	assert.Nil(s.T(), retrieved, "scan data should be nil after deletion")

	// Verify findings are also gone
	findings, err := s.store.GetFindingsByImageID(ctx, imageID)
	require.NoError(s.T(), err)
	assert.Empty(s.T(), findings, "findings should be empty after deletion")
}
