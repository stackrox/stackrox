package service

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/cve/image/datastore/mocks"
	imageDSMocks "github.com/stackrox/rox/central/image/datastore/mocks"
	riskManagerMocks "github.com/stackrox/rox/central/risk/manager/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/enricher"
	enricherMocks "github.com/stackrox/rox/pkg/images/enricher/mocks"
	"github.com/stackrox/rox/pkg/images/utils"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
)

// TestEnrichLocalImageInternal_DeferredCVEsFiltered verifies that suppressed/deferred
// CVEs are properly filtered when returning scan results to Sensor during delegated scanning.
func TestEnrichLocalImageInternal_DeferredCVEsFiltered(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	genImageName := func(img string) *storage.ImageName {
		imgName, _, err := utils.GenerateImageNameFromString(img)
		require.NoError(t, err)
		return imgName
	}

	// Create test CVEs - mix of normal and suppressed
	testCVEs := []*storage.EmbeddedVulnerability{
		{
			Cve:      "CVE-2024-0001",
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			State:    storage.VulnerabilityState_OBSERVED, // Normal CVE
		},
		{
			Cve:      "CVE-2024-0002",
			Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
			State:    storage.VulnerabilityState_OBSERVED, // Will be marked as DEFERRED by suppressor
		},
		{
			Cve:      "CVE-2024-0003",
			Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
			State:    storage.VulnerabilityState_OBSERVED, // Normal CVE
		},
	}

	// Create mock image enricher that returns vulnerabilities
	imageEnricherMock := enricherMocks.NewMockImageEnricher(ctrl)
	imageEnricherMock.EXPECT().
		EnrichWithVulnerabilities(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(img *storage.Image, comps interface{}, notes []scannerV1.Note) (enricher.EnrichmentResult, error) {
			// Simulate Scanner returning CVEs
			img.Scan = &storage.ImageScan{
				Components: []*storage.EmbeddedImageScanComponent{
					{
						Name:    "test-package",
						Version: "1.0.0",
						Vulns:   testCVEs,
					},
				},
			}
			return enricher.EnrichmentResult{ImageUpdated: true}, nil
		})

	imageEnricherMock.EXPECT().
		EnrichWithSignatureVerificationData(gomock.Any(), gomock.Any()).
		AnyTimes().
		Return(enricher.EnrichmentResult{}, nil)

	// Mock image datastore - return existing image with no scan (forces rescan)
	imageDSMock := imageDSMocks.NewMockDataStore(ctrl)
	imageDSMock.EXPECT().
		GetImage(gomock.Any(), gomock.Any()).
		Return(&storage.Image{
			Id:   "test-image-id",
			Scan: nil, // No existing scan
			Names: []*storage.ImageName{
				genImageName("test/image:v1"),
			},
		}, true, nil)

	// Mock risk manager
	riskManagerMock := riskManagerMocks.NewMockManager(ctrl)
	riskManagerMock.EXPECT().
		CalculateRiskAndUpsertImage(gomock.Any()).
		AnyTimes().
		Return(nil)

	// Mock CVE suppressor - this is the KEY part of the test
	cveSuppressorMock := mocks.NewMockDataStore(ctrl)
	cveSuppressorMock.EXPECT().
		EnrichImageWithSuppressedCVEs(gomock.Any()).
		DoAndReturn(func(img *storage.Image) {
			// Simulate marking CVE-2024-0002 as deferred
			for _, comp := range img.GetScan().GetComponents() {
				for _, vuln := range comp.GetVulns() {
					if vuln.GetCve() == "CVE-2024-0002" {
						vuln.Suppressed = true
						vuln.State = storage.VulnerabilityState_DEFERRED
					}
				}
			}
		})

	// Mock CVE suppressor V2 (no-op for this test)
	cveSuppressorV2Mock := mocks.NewMockDataStore(ctrl)
	cveSuppressorV2Mock.EXPECT().
		EnrichImageWithSuppressedCVEs(gomock.Any()).
		AnyTimes().
		Do(func(img *storage.Image) {
			// No-op for V1 images
		})

	// Create service with mocks
	s := serviceImpl{
		internalScanSemaphore: semaphore.NewWeighted(int64(env.MaxParallelImageScanInternal.IntegerSetting())),
		enricher:              imageEnricherMock,
		datastore:             imageDSMock,
		riskManager:           riskManagerMock,
		cveSupressor:          cveSuppressorMock,
		cveSuppressorV2:       cveSuppressorV2Mock,
	}

	// Test WITHOUT fix - comment this to test WITH fix
	// (The actual fix would add calls to cveSupressor.EnrichImageWithSuppressedCVEs)

	// Call EnrichLocalImageInternal
	resp, err := s.EnrichLocalImageInternal(ctx, &v1.EnrichLocalImageInternalRequest{
		ImageId:   "test-image-id",
		ImageName: genImageName("test/image:v1"),
		Components: &scannerV1.Components{
			Namespace: "test",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.GetImage())
	require.NotNil(t, resp.GetImage().GetScan())

	// Verify CVE filtering
	components := resp.GetImage().GetScan().GetComponents()
	require.Len(t, components, 1)

	returnedCVEs := components[0].GetVulns()

	// WITH FIX: Should only have 2 CVEs (CVE-2024-0001 and CVE-2024-0003)
	// WITHOUT FIX: Will have all 3 CVEs

	// Assert expected behavior WITH fix
	assert.Len(t, returnedCVEs, 2, "Deferred CVE should be filtered out")

	cveIDs := make([]string, len(returnedCVEs))
	for i, cve := range returnedCVEs {
		cveIDs[i] = cve.GetCve()
	}

	assert.Contains(t, cveIDs, "CVE-2024-0001", "Normal CVE should be present")
	assert.Contains(t, cveIDs, "CVE-2024-0003", "Normal CVE should be present")
	assert.NotContains(t, cveIDs, "CVE-2024-0002", "Deferred CVE should be filtered")

	// Verify none of the returned CVEs are suppressed
	for _, cve := range returnedCVEs {
		assert.False(t, cve.GetSuppressed(), "Returned CVE %s should not be suppressed", cve.GetCve())
		assert.Equal(t, storage.VulnerabilityState_OBSERVED, cve.GetState(),
			"Returned CVE %s should have OBSERVED state", cve.GetCve())
	}
}
