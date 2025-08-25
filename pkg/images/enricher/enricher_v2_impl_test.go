package enricher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	delegatorMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/integration/mocks"
	imgTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/protoassert"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannertypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/time/rate"
)

func emptyImageGetterV2(_ context.Context, _ string) (*storage.ImageV2, bool, error) {
	return nil, false, nil
}

func imageGetterV2FromImage(image *storage.ImageV2) ImageGetterV2 {
	return func(ctx context.Context, id string) (*storage.ImageV2, bool, error) {
		return image, true, nil
	}
}

func imageGetterV2PanicOnCall(_ context.Context, _ string) (*storage.ImageV2, bool, error) {
	panic("Unexpected call to imageGetter")
}

var _ signatures.SignatureFetcher = (*fakeSigFetcher)(nil)

var _ scannertypes.Scanner = (*fakeScanner)(nil)

var (
	_ scannertypes.ImageScannerWithDataSource = (*fakeRegistryScanner)(nil)
	_ types.ImageRegistry                     = (*fakeRegistryScanner)(nil)
)

func TestEnricherV2Flow(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	cases := []struct {
		name                 string
		ctx                  EnrichmentContext
		inMetadataCache      bool
		shortCircuitRegistry bool
		shortCircuitScanner  bool
		image                *storage.ImageV2
		imageGetter          ImageGetterV2
		fsr                  *fakeRegistryScanner
		result               EnrichmentResult
		errorExpected        bool
	}{
		{
			name: "nothing in the cache",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache: false,
			image: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "reg"},
			},

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: true,
				requestedScan:     true,
			}),
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
			shortCircuitRegistry: false,
			shortCircuitScanner:  true,
			image: &storage.ImageV2{
				Id:  "id",
				Sha: "sha",
			},
			imageGetter: imageGetterV2FromImage(&storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "reg"},
				Scan: &storage.ImageScan{}}),

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     false,
			}),
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
			image: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "reg"},
			},

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: true,
				requestedScan:     true,
			}),
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: " data in both caches but force refetch use names",
			ctx: EnrichmentContext{
				FetchOpt: UseImageNamesRefetchCachedValues,
			},
			inMetadataCache: true,
			image: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "reg"},
			},
			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: true,
				requestedScan:     true,
			}),
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name: "data in both caches but force refetch scans only",
			ctx: EnrichmentContext{
				FetchOpt: ForceRefetchScansOnly,
			},
			inMetadataCache: true,
			image: &storage.ImageV2{
				Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"},
			},

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     true,
			}),
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
		{
			name:          "set ScannerTypeHint to something not found in integrations",
			errorExpected: true,
			ctx: EnrichmentContext{
				FetchOpt:        ForceRefetchScansOnly,
				ScannerTypeHint: "type-test",
			},
			inMetadataCache: true,
			image: &storage.ImageV2{
				Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"},
			},

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     false,
			}),
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanNotDone,
			},
		},
		{
			name: "set ScannerTypeHint to something found in integrations",
			ctx: EnrichmentContext{
				FetchOpt:        ForceRefetchScansOnly,
				ScannerTypeHint: "type",
			},
			inMetadataCache: true,
			image: &storage.ImageV2{
				Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"},
			},

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     true,
			}),
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
			shortCircuitRegistry: true,
			shortCircuitScanner:  true,
			image:                &storage.ImageV2{Id: "id", Sha: "sha"},

			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     false,
			}),
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
			shortCircuitRegistry: false,
			shortCircuitScanner:  true,
			image: &storage.ImageV2{
				Id:       "id",
				Sha:      "sha",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
				Name:     &storage.ImageName{Registry: "reg"},
			},
			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     false,
			}),
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
			image: &storage.ImageV2{
				Id:  "id",
				Sha: "sha",
				Name: &storage.ImageName{
					Registry: "reg",
				},
				Scan: &storage.ImageScan{},
			},
			imageGetter: imageGetterV2PanicOnCall,
			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: true,
				requestedScan:     true,
			}),
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
			shortCircuitRegistry: false,
			shortCircuitScanner:  false,
			image: &storage.ImageV2{
				Id:       "id",
				Sha:      "sha",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
				Name:     &storage.ImageName{Registry: "reg"},
			},
			imageGetter: imageGetterV2PanicOnCall,
			fsr: newFakeRegistryScanner(opts{
				requestedMetadata: false,
				requestedScan:     true,
			}),
			result: EnrichmentResult{
				ImageUpdated: true,
				ScanResult:   ScanSucceeded,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			fsr := newFakeRegistryScanner(opts{})

			registrySet := registryMocks.NewMockSet(ctrl)
			registrySet.EXPECT().Get(gomock.Any()).Return(fsr).AnyTimes()

			set := mocks.NewMockSet(ctrl)
			set.EXPECT().RegistrySet().AnyTimes().Return(registrySet)

			if !c.shortCircuitRegistry {
				registrySet.EXPECT().IsEmpty().AnyTimes().Return(false)
				registrySet.EXPECT().GetAllUnique().AnyTimes().Return([]types.ImageRegistry{fsr})
			}

			scannerSet := scannerMocks.NewMockSet(ctrl)
			if !c.shortCircuitScanner {
				scannerSet.EXPECT().IsEmpty().Return(false)
				scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()
				set.EXPECT().ScannerSet().Return(scannerSet)
			}

			mockReporter := reporterMocks.NewMockReporter(ctrl)
			mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

			enricherImpl := &enricherV2Impl{
				cvesSuppressor:             &fakeCVESuppressorV2{},
				integrations:               set,
				errorsPerScanner:           map[scannertypes.ImageScannerWithDataSource]int32{fsr: 0},
				errorsPerRegistry:          map[types.ImageRegistry]int32{fsr: 0},
				integrationHealthReporter:  mockReporter,
				metadataLimiter:            rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
				metadataCache:              newCache(),
				metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
				imageGetter:                emptyImageGetterV2,
				signatureIntegrationGetter: emptySignatureIntegrationGetter,
				signatureFetcher:           &fakeSigFetcher{},
			}
			if c.inMetadataCache {
				enricherImpl.metadataCache.Add(c.image.GetId(), c.image.GetMetadata())
			}
			if c.imageGetter != nil {
				enricherImpl.imageGetter = c.imageGetter
			}
			result, err := enricherImpl.EnrichImage(emptyCtx, c.ctx, c.image)
			if !c.errorExpected {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			assert.Equal(t, c.result, result)

			assert.Equal(t, c.fsr, fsr)
		})
	}
}

func TestCVESuppressionV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	fsr := newFakeRegistryScanner(opts{})
	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{fsr}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(false)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr})

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := &enricherV2Impl{
		cvesSuppressor:             &fakeCVESuppressorV2{},
		integrations:               set,
		errorsPerScanner:           map[scannertypes.ImageScannerWithDataSource]int32{fsr: 0},
		errorsPerRegistry:          map[types.ImageRegistry]int32{fsr: 0},
		integrationHealthReporter:  mockReporter,
		metadataLimiter:            rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:              newCache(),
		metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
		imageGetter:                emptyImageGetterV2,
		signatureIntegrationGetter: emptySignatureIntegrationGetter,
		signatureFetcher:           &fakeSigFetcher{},
	}

	img := &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	require.NoError(t, err)
	assert.True(t, results.ImageUpdated)
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, img.GetScan().GetComponents()[0].GetVulns()[0].GetState())
}

func TestZeroIntegrationsV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(true).AnyTimes()
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := newEnricherV2(set, mockReporter)

	img := &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: not found: " +
		"no image registries are integrated: please add an image integration"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroIntegrationsInternalV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := newEnricherV2(set, mockReporter)

	img := &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{Internal: true}, img)
	assert.NoError(t, err)
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestRegistryMissingFromImageV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{}).AnyTimes()

	fsr := newFakeRegistryScanner(opts{})
	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().AnyTimes().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := newEnricherV2(set, mockReporter)

	img := &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{FullName: "testimage"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := fmt.Sprintf("image enrichment error: error getting metadata for image: %s "+
		"error: invalid arguments: no registry is indicated for image %q",
		img.GetName().GetFullName(), img.GetName().GetFullName())
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroRegistryIntegrationsV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(true).AnyTimes()
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{}).AnyTimes()

	fsr := newFakeRegistryScanner(opts{})
	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := newEnricherV2(set, mockReporter)

	img := &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: not found: " +
		"no image registries are integrated: please add an image integration"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestNoMatchingRegistryIntegrationV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	fsr := newFakeRegistryScanner(opts{
		notMatch: true,
	})
	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{fsr}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()
	enricherImpl := newEnricherV2(set, mockReporter)

	img := &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: no matching image " +
		"registries found: please add an image integration for reg"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroScannerIntegrationsV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	fsr := newFakeRegistryScanner(opts{})
	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{fsr}).AnyTimes()
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()
	scannerSet.EXPECT().IsEmpty().Return(true)

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()
	enricherImpl := newEnricherV2(set, mockReporter)

	img := &storage.ImageV2{
		Id:   "id",
		Sha:  "sha",
		Name: &storage.ImageName{Registry: "reg"},
	}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error scanning image:  error: no image scanners are integrated"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.True(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestFillScanStatsV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	cases := []struct {
		image                            *storage.ImageV2
		expectedCveCount                 int32
		expectedUnknownCveCount          int32
		expectedFixableUnknownCveCount   int32
		expectedCriticalCveCount         int32
		expectedFixableCriticalCveCount  int32
		expectedImportantCveCount        int32
		expectedFixableImportantCveCount int32
		expectedModerateCveCount         int32
		expectedFixableModerateCveCount  int32
		expectedLowCveCount              int32
		expectedFixableLowCveCount       int32
		expectedFixableCveCount          int32
	}{
		{
			image: &storage.ImageV2{
				Id:  "image-1",
				Sha: "sha",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-1",
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
				},
			},
			expectedCveCount:                 1,
			expectedUnknownCveCount:          0,
			expectedFixableUnknownCveCount:   0,
			expectedCriticalCveCount:         1,
			expectedFixableCriticalCveCount:  1,
			expectedImportantCveCount:        0,
			expectedFixableImportantCveCount: 0,
			expectedModerateCveCount:         0,
			expectedFixableModerateCveCount:  0,
			expectedLowCveCount:              0,
			expectedFixableLowCveCount:       0,
			expectedFixableCveCount:          1,
		},
		{
			image: &storage.ImageV2{
				Id:  "image-1",
				Sha: "sha",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve: "cve-1",
									SetFixedBy: &storage.EmbeddedVulnerability_FixedBy{
										FixedBy: "blah",
									},
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
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
									Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
				},
			},
			expectedCveCount:                 2,
			expectedUnknownCveCount:          1,
			expectedFixableUnknownCveCount:   1,
			expectedCriticalCveCount:         1,
			expectedFixableCriticalCveCount:  1,
			expectedImportantCveCount:        0,
			expectedFixableImportantCveCount: 0,
			expectedModerateCveCount:         0,
			expectedFixableModerateCveCount:  0,
			expectedLowCveCount:              0,
			expectedFixableLowCveCount:       0,
			expectedFixableCveCount:          2,
		},
		{
			image: &storage.ImageV2{
				Id:  "image-1",
				Sha: "sha",
				Scan: &storage.ImageScan{
					Components: []*storage.EmbeddedImageScanComponent{
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-1",
									Severity: storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-2",
									Severity: storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-3",
									Severity: storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-4",
									Severity: storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY,
								},
							},
						},
						{
							Vulns: []*storage.EmbeddedVulnerability{
								{
									Cve:      "cve-5",
									Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY,
								},
							},
						},
					},
				},
			},
			expectedCveCount:                 5,
			expectedUnknownCveCount:          1,
			expectedFixableUnknownCveCount:   0,
			expectedCriticalCveCount:         1,
			expectedFixableCriticalCveCount:  0,
			expectedImportantCveCount:        1,
			expectedFixableImportantCveCount: 0,
			expectedModerateCveCount:         1,
			expectedFixableModerateCveCount:  0,
			expectedLowCveCount:              1,
			expectedFixableLowCveCount:       0,
			expectedFixableCveCount:          0,
		},
	}

	for _, c := range cases {
		t.Run(t.Name(), func(t *testing.T) {
			FillScanStatsV2(c.image)
			assert.Equal(t, c.expectedCveCount, c.image.GetCveCount())
			assert.Equal(t, c.expectedUnknownCveCount, c.image.GetUnknownCveCount())
			assert.Equal(t, c.expectedFixableUnknownCveCount, c.image.GetFixableUnknownCveCount())
			assert.Equal(t, c.expectedCriticalCveCount, c.image.GetCriticalCveCount())
			assert.Equal(t, c.expectedFixableCriticalCveCount, c.image.GetFixableCriticalCveCount())
			assert.Equal(t, c.expectedImportantCveCount, c.image.GetImportantCveCount())
			assert.Equal(t, c.expectedFixableImportantCveCount, c.image.GetFixableImportantCveCount())
			assert.Equal(t, c.expectedModerateCveCount, c.image.GetModerateCveCount())
			assert.Equal(t, c.expectedFixableModerateCveCount, c.image.GetFixableModerateCveCount())
			assert.Equal(t, c.expectedLowCveCount, c.image.GetLowCveCount())
			assert.Equal(t, c.expectedFixableLowCveCount, c.image.GetFixableLowCveCount())
			assert.Equal(t, c.expectedFixableCveCount, c.image.GetFixableCveCount())
		})
	}
}

func TestEnrichWithSignatureV2_Success(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	cases := map[string]struct {
		img                  *storage.ImageV2
		sigFetcher           signatures.SignatureFetcher
		expectedSigs         []*storage.Signature
		updated              bool
		ctx                  EnrichmentContext
		sigIntegrationGetter SignatureIntegrationGetter
	}{
		"signatures found without pre-existing signatures": {
			img: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "reg"},
			},
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:              true,
			sigIntegrationGetter: fakeSignatureIntegrationGetter("test", false),
		},
		"no external metadata enrichment context": {
			ctx: EnrichmentContext{FetchOpt: NoExternalMetadata},
		},
		"cached values should be respected": {
			ctx: EnrichmentContext{FetchOpt: UseCachesIfPossible},
			img: &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}, Signature: &storage.ImageSignature{
				Signatures: []*storage.Signature{createSignature("rawsignature", "rawpayload")}},
			},
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			sigIntegrationGetter: fakeSignatureIntegrationGetter("test", false),
		},
		"fetched signatures contains duplicate": {
			img: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "reg"},
			},
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload"),
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:              true,
			sigIntegrationGetter: fakeSignatureIntegrationGetter("test", false),
		},
		"enrichment should be skipped if no signature integrations available": {
			ctx:                  EnrichmentContext{FetchOpt: NoExternalMetadata},
			sigIntegrationGetter: emptySignatureIntegrationGetter,
		},
		"enrichment should be skipped if only default Red Hat integration available and not Red Hat image": {
			img: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "not-redhat.io"},
			},
			ctx:                  EnrichmentContext{FetchOpt: NoExternalMetadata},
			sigIntegrationGetter: defaultRedHatSignatureIntegrationGetter,
		},
		"enrichment should be performed if only default Red Hat integration available and Red Hat image": {
			img: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "registry.redhat.io"},
			},
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:              true,
			sigIntegrationGetter: defaultRedHatSignatureIntegrationGetter,
		},
		"enrichment should be performed for any image if several integrations available": {
			img: &storage.ImageV2{
				Id:   "id",
				Sha:  "sha",
				Name: &storage.ImageName{Registry: "not-redhat.io"},
			},
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:              true,
			sigIntegrationGetter: twoSignaturesIntegrationGetter,
		},
	}

	ctrl := gomock.NewController(t)
	fsr := newFakeRegistryScanner(opts{})
	registrySetMock := registryMocks.NewMockSet(ctrl)
	registrySetMock.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySetMock.EXPECT().GetAllUnique().Return([]types.ImageRegistry{fsr}).AnyTimes()

	integrationsSetMock := mocks.NewMockSet(ctrl)
	integrationsSetMock.EXPECT().RegistrySet().AnyTimes().Return(registrySetMock)

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherV2Impl{
				integrations:               integrationsSetMock,
				signatureFetcher:           c.sigFetcher,
				signatureIntegrationGetter: c.sigIntegrationGetter,
			}
			updated, err := e.enrichWithSignature(emptyCtx, c.ctx, c.img)
			assert.NoError(t, err)
			assert.Equal(t, c.updated, updated)
			protoassert.ElementsMatch(t, c.expectedSigs, c.img.GetSignature().GetSignatures())
		})
	}
}

func TestEnrichWithSignatureV2_Failures(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctrl := gomock.NewController(t)

	emptyRegistrySetMock := registryMocks.NewMockSet(ctrl)
	emptyRegistrySetMock.EXPECT().IsEmpty().Return(true).AnyTimes()
	emptyRegistrySetMock.EXPECT().GetAllUnique().Return(nil).AnyTimes()

	nonMatchingRegistrySetMock := registryMocks.NewMockSet(ctrl)
	nonMatchingRegistrySetMock.EXPECT().IsEmpty().Return(false).AnyTimes()
	nonMatchingRegistrySetMock.EXPECT().GetAllUnique().Return([]types.ImageRegistry{
		newFakeRegistryScanner(opts{notMatch: true}),
	}).AnyTimes()

	emptyIntegrationSetMock := mocks.NewMockSet(ctrl)
	emptyIntegrationSetMock.EXPECT().RegistrySet().Return(emptyRegistrySetMock).AnyTimes()

	nonMatchingIntegrationSetMock := mocks.NewMockSet(ctrl)
	nonMatchingIntegrationSetMock.EXPECT().RegistrySet().Return(nonMatchingRegistrySetMock).AnyTimes()

	cases := map[string]struct {
		img            *storage.ImageV2
		integrationSet integration.Set
		err            error
	}{
		"no registry set for the image": {
			img: &storage.ImageV2{Id: "id", Sha: "sha"},
			err: errox.InvalidArgs,
		},
		"no registry available": {
			img:            &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}},
			integrationSet: emptyIntegrationSetMock,
			err:            errox.NotFound,
		},
		"no matching registry found": {
			img:            &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{Registry: "reg"}},
			integrationSet: nonMatchingIntegrationSetMock,
			err:            errox.NotFound,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherV2Impl{
				integrations:               c.integrationSet,
				signatureIntegrationGetter: fakeSignatureIntegrationGetter("test", false),
			}
			updated, err := e.enrichWithSignature(emptyCtx,
				EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly}, c.img)
			require.Error(t, err)
			assert.False(t, updated)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestEnrichWithSignatureVerificationDataV2_Success(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	cases := map[string]struct {
		img                         *storage.ImageV2
		sigVerifier                 signatureVerifierForIntegrations
		sigIntegrationGetter        SignatureIntegrationGetter
		expectedVerificationResults []*storage.ImageSignatureVerificationResult
		updated                     bool
		ctx                         EnrichmentContext
	}{
		"verification result found without pre-existing verification results": {
			img: &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{FullName: "test:1.0"}, Signature: &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}},
			sigVerifier: func(ctx context.Context, integrations []*storage.SignatureIntegration, image *storage.Image) []*storage.ImageSignatureVerificationResult {
				return []*storage.ImageSignatureVerificationResult{
					createSignatureVerificationResult("verifier1", storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
				}
			},
			sigIntegrationGetter: fakeSignatureIntegrationGetter("verifier1", false),
			expectedVerificationResults: []*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("verifier1",
					storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
			},
			updated: true,
			ctx:     EnrichmentContext{FetchOpt: ForceRefetch},
		},
		"empty signature integrations without pre-existing verification results": {
			img: &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{FullName: "test:1.0"},
				Signature: &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}},
			sigIntegrationGetter: emptySignatureIntegrationGetter,
			ctx:                  EnrichmentContext{FetchOpt: ForceRefetch},
		},
		"empty signature integration with pre-existing verification results": {
			img: &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{FullName: "test:1.0"},
				Signature: &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}},
				SignatureVerificationData: &storage.ImageSignatureVerificationData{
					Results: []*storage.ImageSignatureVerificationResult{
						createSignatureVerificationResult("verifier1",
							storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
					}}},
			sigIntegrationGetter: emptySignatureIntegrationGetter,
			ctx:                  EnrichmentContext{FetchOpt: UseCachesIfPossible},
			updated:              true,
		},
		"cached values should be respected": {
			img: &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{FullName: "test:1.0"},
				Signature: &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}},
				SignatureVerificationData: &storage.ImageSignatureVerificationData{
					Results: []*storage.ImageSignatureVerificationResult{
						createSignatureVerificationResult("verifier1",
							storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
					}}},
			sigIntegrationGetter: fakeSignatureIntegrationGetter("verifier1", false),
			ctx:                  EnrichmentContext{FetchOpt: UseCachesIfPossible},
			expectedVerificationResults: []*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("verifier1",
					storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
			},
		},
		"no external metadata should be respected": {
			img: &storage.ImageV2{Id: "id", Sha: "sha"},
			ctx: EnrichmentContext{FetchOpt: NoExternalMetadata},
		},
		"empty signature without pre-existing verification results": {
			img: &storage.ImageV2{Id: "id", Sha: "sha"},
		},
		"empty signature with pre-existing verification results": {
			img: &storage.ImageV2{Id: "id", Sha: "sha", Name: &storage.ImageName{FullName: "test:1.0"},
				SignatureVerificationData: &storage.ImageSignatureVerificationData{
					Results: []*storage.ImageSignatureVerificationResult{
						createSignatureVerificationResult("verifier1",
							storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
					}}},
			ctx:     EnrichmentContext{FetchOpt: UseCachesIfPossible},
			updated: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherV2Impl{
				signatureIntegrationGetter: c.sigIntegrationGetter,
				signatureVerifier:          c.sigVerifier,
			}

			updated, err := e.enrichWithSignatureVerificationData(emptyCtx, c.ctx, c.img)
			assert.NoError(t, err)
			assert.Equal(t, c.updated, updated)
			protoassert.ElementsMatch(t, c.expectedVerificationResults, c.img.GetSignatureVerificationData().GetResults())
		})
	}
}

func TestEnrichWithSignatureVerificationDataV2_Failure(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	e := enricherV2Impl{
		signatureIntegrationGetter: fakeSignatureIntegrationGetter("", true),
	}
	img := &storage.ImageV2{Id: "id", Sha: "sha", Signature: &storage.ImageSignature{
		Signatures: []*storage.Signature{createSignature("sig", "pay")},
	}}

	updated, err := e.enrichWithSignatureVerificationData(emptyCtx,
		EnrichmentContext{FetchOpt: ForceRefetch}, img)
	require.Error(t, err)
	assert.False(t, updated)
}

func TestDelegateEnrichImageV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	deleEnrichCtx := EnrichmentContext{Delegable: true}
	e := enricherV2Impl{
		cvesSuppressor: &fakeCVESuppressorV2{},
		imageGetter:    emptyImageGetterV2,
	}

	var dele *delegatorMocks.MockDelegator
	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		dele = delegatorMocks.NewMockDelegator(ctrl)
		e.scanDelegator = dele
	}

	t.Run("not delegable", func(t *testing.T) {
		setup(t)
		enrichCtx := EnrichmentContext{Delegable: false}

		should, err := e.delegateEnrichImage(emptyCtx, enrichCtx, nil)
		assert.False(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate error", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("", false, errBroken)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.False(t, should)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("should not delegate", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("", false, nil)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.False(t, should)
		assert.NoError(t, err)
	})

	t.Run("error should delegate", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("", true, errBroken)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.True(t, should)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("delegate enrich success", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.ImageV2{}
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImageV2(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(fakeImage, nil)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich error", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImageV2(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(nil, errBroken)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.True(t, should)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("delegate enrich cached image", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		img := &storage.ImageV2{
			Id:       "id",
			Sha:      "sha",
			Name:     &storage.ImageName{Registry: "reg"},
			Metadata: &storage.ImageMetadata{},
			Scan:     &storage.ImageScan{},
		}
		e.imageGetter = imageGetterV2FromImage(img)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, img)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich success with cluster id provided", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.ImageV2{}
		dele.EXPECT().ValidateCluster("cluster-id").Return(nil)
		dele.EXPECT().DelegateScanImageV2(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(fakeImage, nil)

		deleEnrichCtx := EnrichmentContext{Delegable: true, ClusterID: "cluster-id"}

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich error with cluster id provided", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.ImageV2{}
		dele.EXPECT().ValidateCluster("cluster-id").Return(errBroken)
		deleEnrichCtx := EnrichmentContext{Delegable: true, ClusterID: "cluster-id"}

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.Error(t, err)
	})
}

func TestEnrichImageV2_Delegate(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	deleEnrichCtx := EnrichmentContext{Delegable: true}
	e := enricherV2Impl{
		cvesSuppressor: &fakeCVESuppressorV2{},
		imageGetter:    emptyImageGetterV2,
	}

	var dele *delegatorMocks.MockDelegator
	setup := func(t *testing.T) {
		ctrl := gomock.NewController(t)
		dele = delegatorMocks.NewMockDelegator(ctrl)
		e.scanDelegator = dele
	}

	t.Run("delegate enrich error", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("", true, errBroken)

		result, err := e.EnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.Equal(t, result.ScanResult, ScanNotDone)
		assert.False(t, result.ImageUpdated)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("delegate enrich success", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.ImageV2{}
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImageV2(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(fakeImage, nil)

		result, err := e.EnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.Equal(t, result.ScanResult, ScanSucceeded)
		assert.True(t, result.ImageUpdated)
		assert.NoError(t, err)
	})
}

func TestFetchFromDatabaseV2_ForceFetch(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImageV2(cimg)
	img.Sha = "some-SHA-for-testing"
	img.Id = "Id"

	e := &enricherV2Impl{
		imageGetter: func(ctx context.Context, id string) (*storage.ImageV2, bool, error) {
			img.Signature = &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("test", "test")}}
			img.SignatureVerificationData = &storage.ImageSignatureVerificationData{Results: []*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("test", storage.ImageSignatureVerificationResult_VERIFIED)}}
			return img, true, nil
		},
	}
	imgFetchedFromDB, exists := e.fetchFromDatabase(context.Background(), img, UseImageNamesRefetchCachedValues)
	assert.False(t, exists)
	protoassert.Equal(t, img.GetName(), imgFetchedFromDB.GetName())
	assert.Nil(t, img.GetSignature())
	assert.Nil(t, img.GetSignatureVerificationData())
}

func TestUpdateImageFromDatabaseV2_Metadata(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	const imageSHA = "some-SHA-for-testing"
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImageV2(cimg)
	img.Sha = imageSHA
	img.Id = "Id"
	metadata := &storage.ImageMetadata{
		V1: nil,
		V2: &storage.V2Metadata{
			Digest: imageSHA,
		},
		Version: 2,
	}
	img.Metadata = metadata

	existingImg := imgTypes.ToImageV2(cimg)
	existingImg.Sha = imageSHA
	existingImg.Id = "Id"

	e := &enricherV2Impl{
		imageGetter: func(_ context.Context, id string) (*storage.ImageV2, bool, error) {
			assert.Equal(t, img.Id, id)
			return existingImg, true, nil
		},
	}

	e.updateImageFromDatabase(context.Background(), img, UseCachesIfPossible)
	assert.Equal(t, imageSHA, img.GetSha())
	protoassert.Equal(t, metadata, img.GetMetadata())
}

func TestMetadataUpToDateV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	t.Run("metadata invalid if is nil", func(t *testing.T) {
		e := &enricherV2Impl{}
		assert.False(t, e.metadataIsValid(nil))
		assert.False(t, e.metadataIsValid(&storage.ImageV2{}))
	})

	t.Run("metadata invalid if datasource points to non-existant integration", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		registrySet := registryMocks.NewMockSet(ctrl)
		registrySet.EXPECT().Get(gomock.Any()).Return(nil) // nil return when integration does not exist

		iiSet := mocks.NewMockSet(ctrl)
		iiSet.EXPECT().RegistrySet().Return(registrySet)

		e := &enricherV2Impl{
			integrations: iiSet,
		}
		img := &storage.ImageV2{
			Metadata: &storage.ImageMetadata{
				DataSource: &storage.DataSource{
					Id: "does-not-exist",
				},
			},
		}
		assert.False(t, e.metadataIsValid(img))
	})

	t.Run("metadata invalid if datasource has mirror", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		registrySet := registryMocks.NewMockSet(ctrl)
		registrySet.EXPECT().Get(gomock.Any()).Return(newFakeRegistryScanner(opts{}))

		iiSet := mocks.NewMockSet(ctrl)
		iiSet.EXPECT().RegistrySet().Return(registrySet)

		e := &enricherV2Impl{
			integrations: iiSet,
		}
		img := &storage.ImageV2{
			Metadata: &storage.ImageMetadata{
				DataSource: &storage.DataSource{
					Mirror: "some fake mirror",
				},
			},
		}
		assert.False(t, e.metadataIsValid(img))
	})

	t.Run("metadata valid if datasouce points to an integration that exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		registrySet := registryMocks.NewMockSet(ctrl)
		registrySet.EXPECT().Get(gomock.Any()).Return(newFakeRegistryScanner(opts{})) // Always find an integration

		iiSet := mocks.NewMockSet(ctrl)
		iiSet.EXPECT().RegistrySet().Return(registrySet)

		e := &enricherV2Impl{
			integrations: iiSet,
		}
		img := &storage.ImageV2{
			Metadata: &storage.ImageMetadata{
				DataSource: &storage.DataSource{
					Id: "exists",
				},
			},
		}
		assert.True(t, e.metadataIsValid(img))
	})
}

func newEnricherV2(set *mocks.MockSet, mockReporter *reporterMocks.MockReporter) ImageEnricherV2 {
	return NewV2(&fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		newCache(),
		emptyImageGetterV2,
		mockReporter, emptySignatureIntegrationGetter, nil)
}
