package enricher

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	delegatorMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/errox"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

func emptyImageGetter(_ context.Context, _ string) (*storage.Image, bool, error) {
	return nil, false, nil
}

func imageGetterFromImage(image *storage.Image) ImageGetter {
	return func(ctx context.Context, id string) (*storage.Image, bool, error) {
		return image, true, nil
	}
}

func imageGetterPanicOnCall(_ context.Context, _ string) (*storage.Image, bool, error) {
	panic("Unexpected call to imageGetter")
}

var _ signatures.SignatureFetcher = (*fakeSigFetcher)(nil)

var _ scannertypes.Scanner = (*fakeScanner)(nil)

var (
	_ scannertypes.ImageScannerWithDataSource = (*fakeRegistryScanner)(nil)
	_ types.ImageRegistry                     = (*fakeRegistryScanner)(nil)
)

func TestEnricherFlow(t *testing.T) {
	cases := []struct {
		name                 string
		ctx                  EnrichmentContext
		inMetadataCache      bool
		shortCircuitRegistry bool
		shortCircuitScanner  bool
		image                *storage.Image
		imageGetter          ImageGetter
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
			image: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),

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
			image: storage.Image_builder{
				Id: "id",
			}.Build(),
			imageGetter: imageGetterFromImage(storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
				Scan:  &storage.ImageScan{}}.Build()),

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
			image: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),

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
			image: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),
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
			image: storage.Image_builder{
				Id: "id", Name: storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),

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
			image: storage.Image_builder{
				Id: "id", Name: storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),

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
			image: storage.Image_builder{
				Id: "id", Name: storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),

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
			image:                storage.Image_builder{Id: "id"}.Build(),

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
			image: storage.Image_builder{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
				Name:     storage.ImageName_builder{Registry: "reg"}.Build(),
				Names:    []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),
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
			image: storage.Image_builder{
				Id: "id",
				Name: storage.ImageName_builder{
					Registry: "reg",
				}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
				Scan:  &storage.ImageScan{},
			}.Build(),
			imageGetter: imageGetterPanicOnCall,
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
			image: storage.Image_builder{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
				Name:     storage.ImageName_builder{Registry: "reg"}.Build(),
				Names:    []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),
			imageGetter: imageGetterPanicOnCall,
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

			enricherImpl := &enricherImpl{
				cvesSuppressor:             &fakeCVESuppressor{},
				cvesSuppressorV2:           &fakeCVESuppressorV2{},
				integrations:               set,
				errorsPerScanner:           map[scannertypes.ImageScannerWithDataSource]int32{fsr: 0},
				errorsPerRegistry:          map[types.ImageRegistry]int32{fsr: 0},
				integrationHealthReporter:  mockReporter,
				metadataLimiter:            rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
				metadataCache:              newCache(),
				metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
				imageGetter:                emptyImageGetter,
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

func TestCVESuppression(t *testing.T) {

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

	enricherImpl := &enricherImpl{
		cvesSuppressor:             &fakeCVESuppressor{},
		cvesSuppressorV2:           &fakeCVESuppressorV2{},
		integrations:               set,
		errorsPerScanner:           map[scannertypes.ImageScannerWithDataSource]int32{fsr: 0},
		errorsPerRegistry:          map[types.ImageRegistry]int32{fsr: 0},
		integrationHealthReporter:  mockReporter,
		metadataLimiter:            rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:              newCache(),
		metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
		imageGetter:                emptyImageGetter,
		signatureIntegrationGetter: emptySignatureIntegrationGetter,
		signatureFetcher:           &fakeSigFetcher{},
	}

	imageName := &storage.ImageName{}
	imageName.SetRegistry("reg")
	imageName2 := &storage.ImageName{}
	imageName2.SetRegistry("reg")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	img.SetNames([]*storage.ImageName{imageName2})
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	require.NoError(t, err)
	assert.True(t, results.ImageUpdated)
	assert.True(t, img.GetScan().GetComponents()[0].GetVulns()[0].GetSuppressed())
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, img.GetScan().GetComponents()[0].GetVulns()[0].GetState())
}

func TestZeroIntegrations(t *testing.T) {
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

	enricherImpl := newEnricher(set, mockReporter)

	imageName := &storage.ImageName{}
	imageName.SetRegistry("reg")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: not found: " +
		"no image registries are integrated: please add an image integration"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroIntegrationsInternal(t *testing.T) {
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := newEnricher(set, mockReporter)

	imageName := &storage.ImageName{}
	imageName.SetRegistry("reg")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{Internal: true}, img)
	assert.NoError(t, err)
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestRegistryMissingFromImage(t *testing.T) {
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

	enricherImpl := newEnricher(set, mockReporter)

	imageName := &storage.ImageName{}
	imageName.SetFullName("testimage")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := fmt.Sprintf("image enrichment error: error getting metadata for image: %s "+
		"error: invalid arguments: no registry is indicated for image %q",
		img.GetName().GetFullName(), img.GetName().GetFullName())
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroRegistryIntegrations(t *testing.T) {
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

	enricherImpl := newEnricher(set, mockReporter)

	imageName := &storage.ImageName{}
	imageName.SetRegistry("reg")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: not found: " +
		"no image registries are integrated: please add an image integration"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestNoMatchingRegistryIntegration(t *testing.T) {
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
	enricherImpl := newEnricher(set, mockReporter)

	imageName := &storage.ImageName{}
	imageName.SetRegistry("reg")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error getting metadata for image:  error: no matching image " +
		"registries found: please add an image integration for reg"
	assert.Equal(t, expectedErrMsg, err.Error())
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestZeroScannerIntegrations(t *testing.T) {
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
	enricherImpl := newEnricher(set, mockReporter)

	imageName := &storage.ImageName{}
	imageName.SetRegistry("reg")
	imageName2 := &storage.ImageName{}
	imageName2.SetRegistry("reg")
	img := &storage.Image{}
	img.SetId("id")
	img.SetName(imageName)
	img.SetNames([]*storage.ImageName{imageName2})
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	assert.Error(t, err)
	expectedErrMsg := "image enrichment error: error scanning image:  error: not found: no image scanners are integrated"
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
			image: storage.Image_builder{
				Id: "image-1",
				Scan: storage.ImageScan_builder{
					Components: []*storage.EmbeddedImageScanComponent{
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve:     "cve-1",
									FixedBy: proto.String("blah"),
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve: "cve-1",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			}.Build(),
			expectedVulns:        1,
			expectedFixableVulns: 1,
		},
		{
			image: storage.Image_builder{
				Id: "image-1",
				Scan: storage.ImageScan_builder{
					Components: []*storage.EmbeddedImageScanComponent{
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve:     "cve-1",
									FixedBy: proto.String("blah"),
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve:     "cve-2",
									FixedBy: proto.String("blah"),
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			}.Build(),
			expectedVulns:        2,
			expectedFixableVulns: 2,
		},
		{
			image: storage.Image_builder{
				Id: "image-1",
				Scan: storage.ImageScan_builder{
					Components: []*storage.EmbeddedImageScanComponent{
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve: "cve-1",
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve: "cve-2",
								}.Build(),
							},
						}.Build(),
						storage.EmbeddedImageScanComponent_builder{
							Vulns: []*storage.EmbeddedVulnerability{
								storage.EmbeddedVulnerability_builder{
									Cve: "cve-3",
								}.Build(),
							},
						}.Build(),
					},
				}.Build(),
			}.Build(),
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

func TestEnrichWithSignature_Success(t *testing.T) {
	cases := map[string]struct {
		img                  *storage.Image
		sigFetcher           signatures.SignatureFetcher
		expectedSigs         []*storage.Signature
		updated              bool
		ctx                  EnrichmentContext
		sigIntegrationGetter SignatureIntegrationGetter
	}{
		"signatures found without pre-existing signatures": {
			img: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),
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
			img: storage.Image_builder{Id: "id", Name: storage.ImageName_builder{Registry: "reg"}.Build(), Signature: storage.ImageSignature_builder{
				Signatures: []*storage.Signature{createSignature("rawsignature", "rawpayload")}}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()},
			}.Build(),
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			sigIntegrationGetter: fakeSignatureIntegrationGetter("test", false),
		},
		"fetched signatures contains duplicate": {
			img: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "reg"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "reg"}.Build()}}.Build(),
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
			img: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "not-redhat.io"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "not-redhat.io"}.Build()},
			}.Build(),
			ctx:                  EnrichmentContext{FetchOpt: NoExternalMetadata},
			sigIntegrationGetter: defaultRedHatSignatureIntegrationGetter,
		},
		"enrichment should be performed if only default Red Hat integration available and Red Hat image": {
			img: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "registry.redhat.io"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "registry.redhat.io"}.Build()},
			}.Build(),
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs:         []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:              true,
			sigIntegrationGetter: defaultRedHatSignatureIntegrationGetter,
		},
		"enrichment should be performed for any image if several integrations available": {
			img: storage.Image_builder{
				Id:    "id",
				Name:  storage.ImageName_builder{Registry: "not-redhat.io"}.Build(),
				Names: []*storage.ImageName{storage.ImageName_builder{Registry: "not-redhat.io"}.Build()},
			}.Build(),
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
			e := enricherImpl{
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

func TestEnrichWithSignature_Failures(t *testing.T) {
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
		img            *storage.Image
		integrationSet integration.Set
		err            error
	}{
		"no registry set for the image": {
			img: storage.Image_builder{Id: "id"}.Build(),
			err: errox.InvalidArgs,
		},
		"no registry available": {
			img:            storage.Image_builder{Id: "id", Name: storage.ImageName_builder{Registry: "reg"}.Build()}.Build(),
			integrationSet: emptyIntegrationSetMock,
			err:            errox.NotFound,
		},
		"no matching registry found": {
			img:            storage.Image_builder{Id: "id", Name: storage.ImageName_builder{Registry: "reg"}.Build()}.Build(),
			integrationSet: nonMatchingIntegrationSetMock,
			err:            errox.NotFound,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherImpl{
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

func TestEnrichWithSignatureVerificationData_Success(t *testing.T) {
	cases := map[string]struct {
		img                         *storage.Image
		sigVerifier                 signatureVerifierForIntegrations
		sigIntegrationGetter        SignatureIntegrationGetter
		expectedVerificationResults []*storage.ImageSignatureVerificationResult
		updated                     bool
		ctx                         EnrichmentContext
	}{
		"verification result found without pre-existing verification results": {
			img: storage.Image_builder{Id: "id", Name: storage.ImageName_builder{FullName: "test:1.0"}.Build(), Signature: storage.ImageSignature_builder{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}.Build()}.Build(),
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
			img: storage.Image_builder{Id: "id", Name: storage.ImageName_builder{FullName: "test:1.0"}.Build(),
				Signature: storage.ImageSignature_builder{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}.Build()}.Build(),
			sigIntegrationGetter: emptySignatureIntegrationGetter,
			ctx:                  EnrichmentContext{FetchOpt: ForceRefetch},
		},
		"empty signature integration with pre-existing verification results": {
			img: storage.Image_builder{Id: "id", Name: storage.ImageName_builder{FullName: "test:1.0"}.Build(),
				Signature: storage.ImageSignature_builder{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}.Build(),
				SignatureVerificationData: storage.ImageSignatureVerificationData_builder{
					Results: []*storage.ImageSignatureVerificationResult{
						createSignatureVerificationResult("verifier1",
							storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
					}}.Build()}.Build(),
			sigIntegrationGetter: emptySignatureIntegrationGetter,
			ctx:                  EnrichmentContext{FetchOpt: UseCachesIfPossible},
			updated:              true,
		},
		"cached values should be respected": {
			img: storage.Image_builder{Id: "id", Name: storage.ImageName_builder{FullName: "test:1.0"}.Build(),
				Signature: storage.ImageSignature_builder{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}.Build(),
				SignatureVerificationData: storage.ImageSignatureVerificationData_builder{
					Results: []*storage.ImageSignatureVerificationResult{
						createSignatureVerificationResult("verifier1",
							storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
					}}.Build()}.Build(),
			sigIntegrationGetter: fakeSignatureIntegrationGetter("verifier1", false),
			ctx:                  EnrichmentContext{FetchOpt: UseCachesIfPossible},
			expectedVerificationResults: []*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("verifier1",
					storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
			},
		},
		"no external metadata should be respected": {
			img: storage.Image_builder{Id: "id"}.Build(),
			ctx: EnrichmentContext{FetchOpt: NoExternalMetadata},
		},
		"empty signature without pre-existing verification results": {
			img: storage.Image_builder{Id: "id"}.Build(),
		},
		"empty signature with pre-existing verification results": {
			img: storage.Image_builder{Id: "id", Name: storage.ImageName_builder{FullName: "test:1.0"}.Build(),
				SignatureVerificationData: storage.ImageSignatureVerificationData_builder{
					Results: []*storage.ImageSignatureVerificationResult{
						createSignatureVerificationResult("verifier1",
							storage.ImageSignatureVerificationResult_VERIFIED, "test:1.0"),
					}}.Build()}.Build(),
			ctx:     EnrichmentContext{FetchOpt: UseCachesIfPossible},
			updated: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherImpl{
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

func TestEnrichWithSignatureVerificationData_Failure(t *testing.T) {
	e := enricherImpl{
		signatureIntegrationGetter: fakeSignatureIntegrationGetter("", true),
	}
	is := &storage.ImageSignature{}
	is.SetSignatures([]*storage.Signature{createSignature("sig", "pay")})
	img := &storage.Image{}
	img.SetId("id")
	img.SetSignature(is)

	updated, err := e.enrichWithSignatureVerificationData(emptyCtx,
		EnrichmentContext{FetchOpt: ForceRefetch}, img)
	require.Error(t, err)
	assert.False(t, updated)
}

func TestDelegateEnrichImage(t *testing.T) {
	deleEnrichCtx := EnrichmentContext{Delegable: true}
	e := enricherImpl{
		cvesSuppressor:   &fakeCVESuppressor{},
		cvesSuppressorV2: &fakeCVESuppressorV2{},
		imageGetter:      emptyImageGetter,
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
		fakeImage := &storage.Image{}
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(fakeImage, nil)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich error", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(nil, errBroken)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.True(t, should)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("delegate enrich cached image", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		imageName := &storage.ImageName{}
		imageName.SetRegistry("reg")
		img := &storage.Image{}
		img.SetId("id")
		img.SetName(imageName)
		img.SetMetadata(&storage.ImageMetadata{})
		img.SetScan(&storage.ImageScan{})
		e.imageGetter = imageGetterFromImage(img)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, img)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich success with cluster id provided", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.Image{}
		dele.EXPECT().ValidateCluster("cluster-id").Return(nil)
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(fakeImage, nil)

		deleEnrichCtx := EnrichmentContext{Delegable: true, ClusterID: "cluster-id"}

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich error with cluster id provided", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.Image{}
		dele.EXPECT().ValidateCluster("cluster-id").Return(errBroken)
		deleEnrichCtx := EnrichmentContext{Delegable: true, ClusterID: "cluster-id"}

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.Error(t, err)
	})
}

func TestEnrichImage_Delegate(t *testing.T) {
	deleEnrichCtx := EnrichmentContext{Delegable: true}
	e := enricherImpl{
		cvesSuppressor:   &fakeCVESuppressor{},
		cvesSuppressorV2: &fakeCVESuppressorV2{},
		imageGetter:      emptyImageGetter,
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
		fakeImage := &storage.Image{}
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", "", gomock.Any()).Return(fakeImage, nil)

		result, err := e.EnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.Equal(t, result.ScanResult, ScanSucceeded)
		assert.True(t, result.ImageUpdated)
		assert.NoError(t, err)
	})
}

func TestFetchFromDatabase_ForceFetch(t *testing.T) {
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImage(cimg)
	img.SetId("some-SHA-for-testing")

	secondImageName, _, err := utils.GenerateImageNameFromString("docker.io/test2")
	require.NoError(t, err)
	e := &enricherImpl{
		imageGetter: func(ctx context.Context, id string) (*storage.Image, bool, error) {
			is := &storage.ImageSignature{}
			is.SetSignatures([]*storage.Signature{createSignature("test", "test")})
			img.SetSignature(is)
			isvd := &storage.ImageSignatureVerificationData{}
			isvd.SetResults([]*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("test", storage.ImageSignatureVerificationResult_VERIFIED)})
			img.SetSignatureVerificationData(isvd)
			img.SetNames(append(img.GetNames(), secondImageName))
			return img, true, nil
		},
	}
	imgFetchedFromDB, exists := e.fetchFromDatabase(context.Background(), img, UseImageNamesRefetchCachedValues)
	assert.False(t, exists)
	protoassert.Equal(t, img.GetName(), imgFetchedFromDB.GetName())
	protoassert.ElementsMatch(t, img.GetNames(), imgFetchedFromDB.GetNames())
	assert.Nil(t, img.GetSignature())
	assert.Nil(t, img.GetSignatureVerificationData())
}

func TestUpdateFromDatabase_ImageNames(t *testing.T) {
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImage(cimg)
	img.SetId("sample-SHA")
	testImageName, _, err := utils.GenerateImageNameFromString(img.GetName().GetFullName())
	require.NoError(t, err)

	cimg, err = utils.GenerateImageFromString("docker.io/test2")
	require.NoError(t, err)
	existingImg := imgTypes.ToImage(cimg)
	existingImg.SetId("sample-SHA")
	existingTestImageName, _, err := utils.GenerateImageNameFromString(existingImg.GetName().GetFullName())
	require.NoError(t, err)

	e := &enricherImpl{
		imageGetter: func(_ context.Context, _ string) (*storage.Image, bool, error) {
			return existingImg, true, nil
		},
	}

	cases := map[string]struct {
		expectedImageNames []*storage.ImageName
		opt                FetchOption
	}{
		"UseCachesIfPossible should retain image names and merge them": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
				existingTestImageName,
			},
			opt: UseCachesIfPossible,
		},
		"NoExternalMetadata should retain image names and merge them": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
				existingTestImageName,
			},
			opt: NoExternalMetadata,
		},
		"IgnoreExistingImages should not retain image names": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
			},
			opt: IgnoreExistingImages,
		},
		"ForceRefetch should not retain image names": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
			},
			opt: ForceRefetch,
		},
		"ForceRefetchScansOnly should retain image names": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
				existingTestImageName,
			},
			opt: ForceRefetchScansOnly,
		},
		"ForceRefetchSignaturesOnly should retain image names": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
				existingTestImageName,
			},
			opt: ForceRefetchSignaturesOnly,
		},
		"ForceRefetchCachedValuesOnly should not retain image names": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
			},
			opt: ForceRefetchCachedValuesOnly,
		},
		"UseImageNamesRefetchCachedValues should retain image names": {
			expectedImageNames: []*storage.ImageName{
				testImageName,
				existingTestImageName,
			},
			opt: UseImageNamesRefetchCachedValues,
		},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			testImg := img.CloneVT()
			_ = e.updateImageFromDatabase(context.Background(), testImg, testCase.opt)
			protoassert.ElementsMatch(t, testImg.GetNames(), testCase.expectedImageNames)
		})
	}
}

func TestUpdateImageFromDatabase_NameChanges(t *testing.T) {
	const imageSHA = "some-SHA-for-testing"
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImage(cimg)
	img.SetId(imageSHA)
	isvd := &storage.ImageSignatureVerificationData{}
	isvd.SetResults([]*storage.ImageSignatureVerificationResult{
		createSignatureVerificationResult("test",
			storage.ImageSignatureVerificationResult_VERIFIED)})
	img.SetSignatureVerificationData(isvd)

	cimg, err = utils.GenerateImageFromString("docker.io/test2")
	require.NoError(t, err)
	existingImg := imgTypes.ToImage(cimg)
	existingImg.SetId(imageSHA)

	e := &enricherImpl{
		imageGetter: func(_ context.Context, id string) (*storage.Image, bool, error) {
			isvd2 := &storage.ImageSignatureVerificationData{}
			isvd2.SetResults([]*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("test2",
					storage.ImageSignatureVerificationResult_VERIFIED)})
			existingImg.SetSignatureVerificationData(isvd2)
			return existingImg, true, nil
		},
	}
	e.updateImageFromDatabase(context.Background(), img, UseCachesIfPossible)
	assert.Equal(t, imageSHA, img.GetId())
	// Changes to names should lead to discarding any previously found signature verification results, even if the
	// fetch option indicates to use caches.
	assert.Empty(t, img.GetSignatureVerificationData().GetResults())
}

func TestUpdateImageFromDatabase_Metadata(t *testing.T) {
	const imageSHA = "some-SHA-for-testing"
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImage(cimg)
	img.SetId(imageSHA)
	v2m := &storage.V2Metadata{}
	v2m.SetDigest(imageSHA)
	metadata := &storage.ImageMetadata{}
	metadata.ClearV1()
	metadata.SetV2(v2m)
	metadata.SetVersion(2)
	img.SetMetadata(metadata)

	existingImg := imgTypes.ToImage(cimg)
	existingImg.SetId(imageSHA)

	e := &enricherImpl{
		imageGetter: func(_ context.Context, id string) (*storage.Image, bool, error) {
			assert.Equal(t, imageSHA, id)
			return existingImg, true, nil
		},
	}

	e.updateImageFromDatabase(context.Background(), img, UseCachesIfPossible)
	assert.Equal(t, imageSHA, img.GetId())
	protoassert.Equal(t, metadata, img.GetMetadata())
}

func TestMetadataUpToDate(t *testing.T) {
	t.Run("metadata invalid if is nil", func(t *testing.T) {
		e := &enricherImpl{}
		assert.False(t, e.metadataIsValid(nil))
		assert.False(t, e.metadataIsValid(&storage.Image{}))
	})

	t.Run("metadata invalid if datasource points to non-existant integration", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		registrySet := registryMocks.NewMockSet(ctrl)
		registrySet.EXPECT().Get(gomock.Any()).Return(nil) // nil return when integration does not exist

		iiSet := mocks.NewMockSet(ctrl)
		iiSet.EXPECT().RegistrySet().Return(registrySet)

		e := &enricherImpl{
			integrations: iiSet,
		}
		ds := &storage.DataSource{}
		ds.SetId("does-not-exist")
		im := &storage.ImageMetadata{}
		im.SetDataSource(ds)
		img := &storage.Image{}
		img.SetMetadata(im)
		assert.False(t, e.metadataIsValid(img))
	})

	t.Run("metadata invalid if datasource has mirror", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		registrySet := registryMocks.NewMockSet(ctrl)
		registrySet.EXPECT().Get(gomock.Any()).Return(newFakeRegistryScanner(opts{}))

		iiSet := mocks.NewMockSet(ctrl)
		iiSet.EXPECT().RegistrySet().Return(registrySet)

		e := &enricherImpl{
			integrations: iiSet,
		}
		ds := &storage.DataSource{}
		ds.SetMirror("some fake mirror")
		im := &storage.ImageMetadata{}
		im.SetDataSource(ds)
		img := &storage.Image{}
		img.SetMetadata(im)
		assert.False(t, e.metadataIsValid(img))
	})

	t.Run("metadata valid if datasouce points to an integration that exists", func(t *testing.T) {
		ctrl := gomock.NewController(t)

		registrySet := registryMocks.NewMockSet(ctrl)
		registrySet.EXPECT().Get(gomock.Any()).Return(newFakeRegistryScanner(opts{})) // Always find an integration

		iiSet := mocks.NewMockSet(ctrl)
		iiSet.EXPECT().RegistrySet().Return(registrySet)

		e := &enricherImpl{
			integrations: iiSet,
		}
		ds := &storage.DataSource{}
		ds.SetId("exists")
		im := &storage.ImageMetadata{}
		im.SetDataSource(ds)
		img := &storage.Image{}
		img.SetMetadata(im)
		assert.True(t, e.metadataIsValid(img))
	})
}

func newEnricher(set *mocks.MockSet, mockReporter *reporterMocks.MockReporter) ImageEnricher {
	return New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		newCache(),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)
}
