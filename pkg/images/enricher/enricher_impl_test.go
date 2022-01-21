package enricher

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration/mocks"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	mocks2 "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	mocks3 "github.com/stackrox/rox/pkg/scanners/mocks"
	scannertypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

type fakeRegistryScanner struct {
	requestedMetadata bool
	requestedScan     bool
	notMatch          bool
}

func (f *fakeRegistryScanner) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	f.requestedMetadata = true
	return &storage.ImageMetadata{}, nil
}

func (f *fakeRegistryScanner) Global() bool {
	return true
}

func (f *fakeRegistryScanner) Config() *types.Config {
	return nil
}

func (f *fakeRegistryScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeRegistryScanner) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	f.requestedScan = true
	return &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Vulns: []*storage.EmbeddedVulnerability{
					{
						Cve: "CVE-2020-1234",
					},
				},
			},
		},
	}, nil
}

func (f *fakeRegistryScanner) Match(image *storage.ImageName) bool {
	return !f.notMatch
}

func (f *fakeRegistryScanner) Test() error {
	return nil
}

func (f *fakeRegistryScanner) Type() string {
	return "type"
}

func (f *fakeRegistryScanner) Name() string {
	return "name"
}

func (f *fakeRegistryScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{
		Id:   "id",
		Name: f.Name(),
	}
}

func (f *fakeRegistryScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

type fakeCVESuppressor struct{}

func (f *fakeCVESuppressor) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.Cve == "CVE-2020-1234" {
				v.Suppressed = true
			}
		}
	}
}

type fakeCVESuppressorV2 struct{}

func (f *fakeCVESuppressorV2) EnrichImageWithSuppressedCVEs(image *storage.Image) {
	for _, c := range image.GetScan().GetComponents() {
		for _, v := range c.GetVulns() {
			if v.Cve == "CVE-2020-1234" {
				v.State = storage.VulnerabilityState_DEFERRED
			}
		}
	}
}

func TestEnricherFlow(t *testing.T) {
	cases := []struct {
		name                 string
		ctx                  EnrichmentContext
		inMetadataCache      bool
		inScanCache          bool
		shortCircuitRegistry bool
		shortCircuitScanner  bool
		image                *storage.Image

		fsr    *fakeRegistryScanner
		result EnrichmentResult
	}{
		{
			name: "nothing in the cache",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache: false,
			inScanCache:     false,
			image:           &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}},

			fsr: &fakeRegistryScanner{
				requestedMetadata: true,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "scan and metadata in both caches",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache:      true,
			inScanCache:          true,
			shortCircuitRegistry: true,
			shortCircuitScanner:  true,
			image:                &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in both caches, but force refetch",
			ctx: EnrichmentContext{
				FetchOpt: ForceRefetch,
			},
			inMetadataCache: true,
			inScanCache:     true,
			image:           &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}},

			fsr: &fakeRegistryScanner{
				requestedMetadata: true,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in both caches but force refetch scans only",
			ctx: EnrichmentContext{
				FetchOpt: ForceRefetch,
			},
			inMetadataCache:      true,
			inScanCache:          true,
			shortCircuitRegistry: true,
			image:                &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data not in caches, and no external metadata",
			ctx: EnrichmentContext{
				FetchOpt: NoExternalMetadata,
			},
			inMetadataCache:      false,
			inScanCache:          false,
			shortCircuitRegistry: true,
			shortCircuitScanner:  true,
			image:                &storage.Image{Id: "id"},

			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: false,
				ScanResult:   ScanNotDone,
			},
		},
		{
			name: "data not in cache, but image already has metadata and scan",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache:      false,
			inScanCache:          false,
			shortCircuitRegistry: true,
			shortCircuitScanner:  true,
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
			},
			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: false,
				ScanResult:   ScanNotDone,
			},
		},
		{
			name: "data not in cache and ignore existing images",
			ctx: EnrichmentContext{
				FetchOpt: IgnoreExistingImages,
			},
			inMetadataCache: false,
			inScanCache:     false,
			image: &storage.Image{
				Id: "id",
				Name: &storage.ImageName{
					Registry: "reg",
				},
				Scan: &storage.ImageScan{},
			},
			fsr: &fakeRegistryScanner{
				requestedMetadata: true,
				requestedScan:     true,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in cache and ignore existing images",
			ctx: EnrichmentContext{
				FetchOpt: IgnoreExistingImages,
			},
			inMetadataCache:      true,
			inScanCache:          true,
			shortCircuitRegistry: true,
			shortCircuitScanner:  true,
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
			},
			fsr: &fakeRegistryScanner{
				requestedMetadata: false,
				requestedScan:     false,
			},
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			set := mocks.NewMockSet(ctrl)

			fsr := &fakeRegistryScanner{}
			registrySet := mocks2.NewMockSet(ctrl)
			if !c.shortCircuitRegistry {
				registrySet.EXPECT().IsEmpty().Return(false)
				registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr})
				set.EXPECT().RegistrySet().Return(registrySet)
			}

			scannerSet := mocks3.NewMockSet(ctrl)
			if !c.shortCircuitScanner {
				scannerSet.EXPECT().IsEmpty().Return(false)
				scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{fsr})
				set.EXPECT().ScannerSet().Return(scannerSet)
			}

			mockReporter := reporterMocks.NewMockReporter(ctrl)
			mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

			enricherImpl := &enricherImpl{
				cvesSuppressor:            &fakeCVESuppressor{},
				cvesSuppressorV2:          &fakeCVESuppressorV2{},
				integrations:              set,
				errorsPerScanner:          map[scannertypes.ImageScanner]int32{fsr: 0},
				errorsPerRegistry:         map[types.ImageRegistry]int32{fsr: 0},
				integrationHealthReporter: mockReporter,
				metadataLimiter:           rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
				metrics:                   newMetrics(pkgMetrics.CentralSubsystem),
			}

			if c.inMetadataCache {
				enricherImpl.metadataCache.Add(c.image.GetId(), c.image.GetMetadata())
			}
			if c.inScanCache {
				enricherImpl.scanCache.Add(c.image.GetId(), c.image.GetScan())
			}
			result, err := enricherImpl.EnrichImage(c.ctx, c.image)
			require.NoError(t, err)
			assert.Equal(t, c.result, result)

			assert.Equal(t, c.fsr, fsr)
		})
	}
}

func TestCVESuppression(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fsr := &fakeRegistryScanner{}
	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr})

	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(false)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{fsr})

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet)
	set.EXPECT().ScannerSet().Return(scannerSet)

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := &enricherImpl{
		cvesSuppressor:            &fakeCVESuppressor{},
		cvesSuppressorV2:          &fakeCVESuppressorV2{},
		integrations:              set,
		errorsPerScanner:          map[scannertypes.ImageScanner]int32{fsr: 0},
		errorsPerRegistry:         map[types.ImageRegistry]int32{fsr: 0},
		integrationHealthReporter: mockReporter,
		metadataLimiter:           rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:             expiringcache.NewExpiringCache(1 * time.Minute),
		metrics:                   newMetrics(pkgMetrics.CentralSubsystem),
	}

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{}, img)
	require.NoError(t, err)
	assert.True(t, results.ImageUpdated)
	assert.True(t, img.Scan.Components[0].Vulns[0].Suppressed)
	if features.VulnRiskManagement.Enabled() {
		assert.Equal(t, storage.VulnerabilityState_DEFERRED, img.Scan.Components[0].Vulns[0].State)
	}
}

func TestZeroIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(true)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(true)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		expiringcache.NewExpiringCache(1*time.Minute),
		mockReporter)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment errors: [error getting metadata for image:  error: no image registries are integrated: please add an image integration for reg, error scanning image:  error: no image scanners are integrated]"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroIntegrationsInternal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		expiringcache.NewExpiringCache(1*time.Minute),
		mockReporter)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{Internal: true}, img)
	assert.NoError(t, err)
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestRegistryMissingFromImage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	fsr := &fakeRegistryScanner{}
	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(false)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		expiringcache.NewExpiringCache(1*time.Minute),
		mockReporter)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "testimage"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image: testimage error: no registry is indicated for image"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.True(t, results.ImageUpdated)
	assert.Equal(t, ScanSucceeded, results.ScanResult)
}

func TestZeroRegistryIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(true)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	fsr := &fakeRegistryScanner{}
	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(false)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		expiringcache.NewExpiringCache(1*time.Minute),
		mockReporter)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: no image registries are integrated: please add an image integration for reg"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.True(t, results.ImageUpdated)
	assert.Equal(t, ScanSucceeded, results.ScanResult)
}

func TestNoMatchingRegistryIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fsr := &fakeRegistryScanner{
		notMatch: true,
	}
	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr}).AnyTimes()

	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(false)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()
	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		expiringcache.NewExpiringCache(1*time.Minute),
		mockReporter)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: no matching image registries found: please add an image integration for reg"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroScannerIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fsr := &fakeRegistryScanner{}
	registrySet := mocks2.NewMockSet(ctrl)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr}).AnyTimes()
	registrySet.EXPECT().IsEmpty().Return(false)

	scannerSet := mocks3.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScanner{}).AnyTimes()
	scannerSet.EXPECT().IsEmpty().Return(true)

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()
	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem, nil, mockReporter)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error scanning image:  error: no image scanners are integrated"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.True(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestFillScanStats(t *testing.T) {
	cases := []struct {
		image                *storage.Image
		expectedVulns        int32
		expectedFixableVulns int32
	}{
		{
			image: &storage.Image{
				Id: "image-1",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
								},
							},
						},
					},
				},
			},
			expectedVulns:        1,
			expectedFixableVulns: 1,
		},
		{
			image: &storage.Image{
				Id: "image-1",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
								},
							},
						},
					},
				},
			},
			expectedVulns:        2,
			expectedFixableVulns: 2,
		},
		{
			image: &storage.Image{
				Id: "image-1",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-2",
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-3",
								},
							},
						},
					},
				},
			},
			expectedVulns:        3,
			expectedFixableVulns: 0,
		},
	}

	for _, c := range cases {
		t.Run(t.Name(), func(t *testing.T) {
			FillScanStats(c.image)
			assert.Equal(t, c.expectedVulns, c.image.GetCves())
			assert.Equal(t, c.expectedFixableVulns, c.image.GetFixableCves())
		})
	}
}
