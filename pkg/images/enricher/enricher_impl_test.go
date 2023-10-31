package enricher

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	delegatorMocks "github.com/stackrox/rox/pkg/delegatedregistry/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/integration/mocks"
	imgTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannertypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

var (
	// emptyCtx used within all tests.
	emptyCtx = context.Background()

	// errBroken is a generic error.
	errBroken = errors.New("broken")
)

func emptyImageGetter(_ context.Context, _ string) (*storage.Image, bool, error) {
	return nil, false, nil
}

func emptySignatureIntegrationGetter(_ context.Context) ([]*storage.SignatureIntegration, error) {
	return nil, nil
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

type fakeSigFetcher struct {
	sigs      []*storage.Signature
	fail      bool
	retryable bool
}

func (f *fakeSigFetcher) FetchSignatures(_ context.Context, _ *storage.Image, _ string,
	_ types.Registry) ([]*storage.Signature, error) {
	if f.fail {
		err := errors.New("some error")
		if f.retryable {
			err = retry.MakeRetryable(err)
		}
		return nil, err
	}
	return f.sigs, nil
}

var _ scannertypes.Scanner = (*fakeScanner)(nil)

type fakeScanner struct {
	requestedScan bool
	notMatch      bool
}

func (*fakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return semaphore.NewWeighted(1)
}

func (f *fakeScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
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

func (f *fakeScanner) Match(*storage.ImageName) bool {
	return !f.notMatch
}

func (*fakeScanner) Test() error {
	return nil
}

func (*fakeScanner) Type() string {
	return "type"
}

func (*fakeScanner) Name() string {
	return "name"
}

func (*fakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

var (
	_ scannertypes.ImageScannerWithDataSource = (*fakeRegistryScanner)(nil)
	_ types.ImageRegistry                     = (*fakeRegistryScanner)(nil)
)

type fakeRegistryScanner struct {
	scanner           scannertypes.Scanner
	requestedMetadata bool
	notMatch          bool
}

type opts struct {
	requestedScan     bool
	requestedMetadata bool
	notMatch          bool
}

func newFakeRegistryScanner(opts opts) *fakeRegistryScanner {
	return &fakeRegistryScanner{
		scanner: &fakeScanner{
			requestedScan: opts.requestedScan,
			notMatch:      opts.notMatch,
		},
		requestedMetadata: opts.requestedMetadata,
		notMatch:          opts.notMatch,
	}
}

func (f *fakeRegistryScanner) Metadata(*storage.Image) (*storage.ImageMetadata, error) {
	f.requestedMetadata = true
	return &storage.ImageMetadata{}, nil
}

func (f *fakeRegistryScanner) Config() *types.Config {
	return nil
}

func (f *fakeRegistryScanner) Match(*storage.ImageName) bool {
	return !f.notMatch
}

func (*fakeRegistryScanner) Test() error {
	return nil
}

func (*fakeRegistryScanner) Type() string {
	return "type"
}

func (*fakeRegistryScanner) Name() string {
	return "name"
}

func (*fakeRegistryScanner) HTTPClient() *http.Client {
	return nil
}

func (f *fakeRegistryScanner) GetScanner() scannertypes.Scanner {
	return f.scanner
}

func (f *fakeRegistryScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{
		Id:   "id",
		Name: f.Name(),
	}
}

func (f *fakeRegistryScanner) Source() *storage.ImageIntegration {
	return &storage.ImageIntegration{
		Id:   "id",
		Name: f.Name(),
	}
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
		shortCircuitRegistry bool
		shortCircuitScanner  bool
		image                *storage.Image
		imageGetter          ImageGetter
		fsr                  *fakeRegistryScanner
		result               EnrichmentResult
	}{
		{
			name: "nothing in the cache",
			ctx: EnrichmentContext{
				FetchOpt: UseCachesIfPossible,
			},
			inMetadataCache: false,
			image: &storage.Image{
				Id:    "id",
				Name:  &storage.ImageName{Registry: "reg"},
				Names: []*storage.ImageName{{Registry: "reg"}},
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
			image: &storage.Image{
				Id:    "id",
				Name:  &storage.ImageName{Registry: "reg"},
				Names: []*storage.ImageName{{Registry: "reg"}},
			},
			imageGetter: imageGetterFromImage(&storage.Image{Id: "id", Scan: &storage.ImageScan{}}),

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
			image: &storage.Image{
				Id:    "id",
				Name:  &storage.ImageName{Registry: "reg"},
				Names: []*storage.ImageName{{Registry: "reg"}},
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
			image: &storage.Image{
				Id: "id", Name: &storage.ImageName{Registry: "reg"},
				Names: []*storage.ImageName{{Registry: "reg"}},
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
			image:                &storage.Image{Id: "id"},

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
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
				Name:     &storage.ImageName{Registry: "reg"},
				Names:    []*storage.ImageName{{Registry: "reg"}},
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
			image: &storage.Image{
				Id: "id",
				Name: &storage.ImageName{
					Registry: "reg",
				},
				Names: []*storage.ImageName{{Registry: "reg"}},
				Scan:  &storage.ImageScan{},
			},
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
			image: &storage.Image{
				Id:       "id",
				Metadata: &storage.ImageMetadata{},
				Scan:     &storage.ImageScan{},
				Name:     &storage.ImageName{Registry: "reg"},
				Names:    []*storage.ImageName{{Registry: "reg"}},
			},
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

			set := mocks.NewMockSet(ctrl)

			fsr := newFakeRegistryScanner(opts{})
			registrySet := registryMocks.NewMockSet(ctrl)
			if !c.shortCircuitRegistry {
				registrySet.EXPECT().IsEmpty().AnyTimes().Return(false)
				registrySet.EXPECT().GetAll().AnyTimes().Return([]types.ImageRegistry{fsr})
				set.EXPECT().RegistrySet().AnyTimes().Return(registrySet)
			}

			scannerSet := scannerMocks.NewMockSet(ctrl)
			if !c.shortCircuitScanner {
				scannerSet.EXPECT().IsEmpty().Return(false)
				scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr})
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
				metadataCache:              expiringcache.NewExpiringCache(1 * time.Minute),
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
			require.NoError(t, err)
			assert.Equal(t, c.result, result)

			assert.Equal(t, c.fsr, fsr)
		})
	}
}

func TestCVESuppression(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	fsr := newFakeRegistryScanner(opts{})
	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr}).AnyTimes()

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
		metadataCache:              expiringcache.NewExpiringCache(1 * time.Minute),
		metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
		imageGetter:                emptyImageGetter,
		signatureIntegrationGetter: emptySignatureIntegrationGetter,
		signatureFetcher:           &fakeSigFetcher{},
	}

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"},
		Names: []*storage.ImageName{{Registry: "reg"}}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
	require.NoError(t, err)
	assert.True(t, results.ImageUpdated)
	assert.True(t, img.Scan.Components[0].Vulns[0].Suppressed)
	assert.Equal(t, storage.VulnerabilityState_DEFERRED, img.Scan.Components[0].Vulns[0].State)
}

func TestZeroIntegrations(t *testing.T) {
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(true).AnyTimes()
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
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
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{Internal: true}, img)
	assert.NoError(t, err)
	assert.False(t, results.ImageUpdated)
	assert.Equal(t, ScanNotDone, results.ScanResult)
}

func TestRegistryMissingFromImage(t *testing.T) {
	ctrl := gomock.NewController(t)

	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	fsr := newFakeRegistryScanner(opts{})
	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().AnyTimes().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "testimage"}}
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
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{}).AnyTimes()

	fsr := newFakeRegistryScanner(opts{})
	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
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
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr}).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{fsr}).AnyTimes()

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()
	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)

	img := &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}}
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
	registrySet.EXPECT().GetAll().Return([]types.ImageRegistry{fsr}).AnyTimes()
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()

	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{}).AnyTimes()
	scannerSet.EXPECT().IsEmpty().Return(true)

	set := mocks.NewMockSet(ctrl)
	set.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	set.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()
	enricherImpl := New(&fakeCVESuppressor{}, &fakeCVESuppressorV2{}, set, pkgMetrics.CentralSubsystem,
		expiringcache.NewExpiringCache(1*time.Minute),
		emptyImageGetter,
		mockReporter, emptySignatureIntegrationGetter, nil)

	img := &storage.Image{
		Id:    "id",
		Name:  &storage.ImageName{Registry: "reg"},
		Names: []*storage.ImageName{{Registry: "reg"}},
	}
	results, err := enricherImpl.EnrichImage(emptyCtx, EnrichmentContext{}, img)
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

func TestEnrichWithSignature_Success(t *testing.T) {
	cases := map[string]struct {
		img          *storage.Image
		sigFetcher   signatures.SignatureFetcher
		expectedSigs []*storage.Signature
		updated      bool
		ctx          EnrichmentContext
	}{
		"signatures found without pre-existing signatures": {
			img: &storage.Image{
				Id:    "id",
				Name:  &storage.ImageName{Registry: "reg"},
				Names: []*storage.ImageName{{Registry: "reg"}},
			},
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs: []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:      true,
		},
		"no external metadata enrichment context": {
			ctx: EnrichmentContext{FetchOpt: NoExternalMetadata},
		},
		"cached values should be respected": {
			ctx: EnrichmentContext{FetchOpt: UseCachesIfPossible},
			img: &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}, Signature: &storage.ImageSignature{
				Signatures: []*storage.Signature{createSignature("rawsignature", "rawpayload")}},
				Names: []*storage.ImageName{{Registry: "reg"}},
			},
			expectedSigs: []*storage.Signature{createSignature("rawsignature", "rawpayload")},
		},
		"fetched signatures contains duplicate": {
			img: &storage.Image{
				Id:    "id",
				Name:  &storage.ImageName{Registry: "reg"},
				Names: []*storage.ImageName{{Registry: "reg"}}},
			ctx: EnrichmentContext{FetchOpt: ForceRefetchSignaturesOnly},
			sigFetcher: &fakeSigFetcher{sigs: []*storage.Signature{
				createSignature("rawsignature", "rawpayload"),
				createSignature("rawsignature", "rawpayload")}},
			expectedSigs: []*storage.Signature{createSignature("rawsignature", "rawpayload")},
			updated:      true,
		},
	}

	ctrl := gomock.NewController(t)
	fsr := newFakeRegistryScanner(opts{})
	registrySetMock := registryMocks.NewMockSet(ctrl)
	registrySetMock.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySetMock.EXPECT().GetAll().Return([]types.ImageRegistry{fsr}).AnyTimes()

	integrationsSetMock := mocks.NewMockSet(ctrl)
	integrationsSetMock.EXPECT().RegistrySet().AnyTimes().Return(registrySetMock)

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherImpl{
				integrations:     integrationsSetMock,
				signatureFetcher: c.sigFetcher,
			}
			updated, err := e.enrichWithSignature(emptyCtx, c.ctx, c.img)
			assert.NoError(t, err)
			assert.Equal(t, c.updated, updated)
			assert.ElementsMatch(t, c.expectedSigs, c.img.GetSignature().GetSignatures())
		})
	}
}

func TestEnrichWithSignature_Failures(t *testing.T) {
	ctrl := gomock.NewController(t)

	emptyRegistrySetMock := registryMocks.NewMockSet(ctrl)
	emptyRegistrySetMock.EXPECT().IsEmpty().Return(true).AnyTimes()
	emptyRegistrySetMock.EXPECT().GetAll().Return(nil).AnyTimes()

	nonMatchingRegistrySetMock := registryMocks.NewMockSet(ctrl)
	nonMatchingRegistrySetMock.EXPECT().IsEmpty().Return(false).AnyTimes()
	nonMatchingRegistrySetMock.EXPECT().GetAll().Return([]types.ImageRegistry{
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
			img: &storage.Image{Id: "id"},
			err: errox.InvalidArgs,
		},
		"no registry available": {
			img:            &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}},
			integrationSet: emptyIntegrationSetMock,
			err:            errox.NotFound,
		},
		"no matching registry found": {
			img:            &storage.Image{Id: "id", Name: &storage.ImageName{Registry: "reg"}},
			integrationSet: nonMatchingIntegrationSetMock,
			err:            errox.NotFound,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := enricherImpl{
				integrations: c.integrationSet,
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
			img: &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "test:1.0"}, Signature: &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}},
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
			img: &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "test:1.0"},
				Signature: &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("sig1", "payload1")}}},
			sigIntegrationGetter: emptySignatureIntegrationGetter,
			ctx:                  EnrichmentContext{FetchOpt: ForceRefetch},
		},
		"empty signature integration with pre-existing verification results": {
			img: &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "test:1.0"},
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
			img: &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "test:1.0"},
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
			img: &storage.Image{Id: "id"},
			ctx: EnrichmentContext{FetchOpt: NoExternalMetadata},
		},
		"empty signature without pre-existing verification results": {
			img: &storage.Image{Id: "id"},
		},
		"empty signature with pre-existing verification results": {
			img: &storage.Image{Id: "id", Name: &storage.ImageName{FullName: "test:1.0"},
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
			e := enricherImpl{
				signatureIntegrationGetter: c.sigIntegrationGetter,
				signatureVerifier:          c.sigVerifier,
			}

			updated, err := e.enrichWithSignatureVerificationData(emptyCtx, c.ctx, c.img)
			assert.NoError(t, err)
			assert.Equal(t, c.updated, updated)
			assert.ElementsMatch(t, c.expectedVerificationResults, c.img.GetSignatureVerificationData().GetResults())
		})
	}
}

func TestEnrichWithSignatureVerificationData_Failure(t *testing.T) {
	e := enricherImpl{
		signatureIntegrationGetter: fakeSignatureIntegrationGetter("", true),
	}
	img := &storage.Image{Id: "id", Signature: &storage.ImageSignature{
		Signatures: []*storage.Signature{createSignature("sig", "pay")},
	}}

	updated, err := e.enrichWithSignatureVerificationData(emptyCtx,
		EnrichmentContext{FetchOpt: ForceRefetch}, img)
	require.Error(t, err)
	assert.False(t, updated)
}
func createSignature(sig, payload string) *storage.Signature {
	return &storage.Signature{Signature: &storage.Signature_Cosign{
		Cosign: &storage.CosignSignature{
			RawSignature:     []byte(sig),
			SignaturePayload: []byte(payload),
		},
	}}
}

func createSignatureVerificationResult(verifier string, status storage.ImageSignatureVerificationResult_Status,
	verifiedImageNames ...string) *storage.ImageSignatureVerificationResult {
	return &storage.ImageSignatureVerificationResult{
		VerifierId:              verifier,
		Status:                  status,
		VerifiedImageReferences: verifiedImageNames,
	}
}

func fakeSignatureIntegrationGetter(id string, fail bool) SignatureIntegrationGetter {
	return func(ctx context.Context) ([]*storage.SignatureIntegration, error) {
		if fail {
			return nil, errors.New("fake error")
		}
		return []*storage.SignatureIntegration{
			{
				Id:   id,
				Name: id,
			},
		}, nil
	}
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
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", gomock.Any()).Return(fakeImage, nil)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, fakeImage)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich error", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", gomock.Any()).Return(nil, errBroken)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, nil)
		assert.True(t, should)
		assert.ErrorIs(t, err, errBroken)
	})

	t.Run("delegate enrich cached image", func(t *testing.T) {
		setup(t)
		dele.EXPECT().GetDelegateClusterID(emptyCtx, gomock.Any()).Return("cluster-id", true, nil)
		img := &storage.Image{
			Id:       "id",
			Name:     &storage.ImageName{Registry: "reg"},
			Metadata: &storage.ImageMetadata{},
			Scan:     &storage.ImageScan{},
		}
		e.imageGetter = imageGetterFromImage(img)

		should, err := e.delegateEnrichImage(emptyCtx, deleEnrichCtx, img)
		assert.True(t, should)
		assert.NoError(t, err)
	})

	t.Run("delegate enrich success with cluster id provided", func(t *testing.T) {
		setup(t)
		fakeImage := &storage.Image{}
		dele.EXPECT().ValidateCluster("cluster-id").Return(nil)
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", gomock.Any()).Return(fakeImage, nil)

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
		dele.EXPECT().DelegateScanImage(emptyCtx, gomock.Any(), "cluster-id", gomock.Any()).Return(fakeImage, nil)

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
	img.Id = "some-SHA-for-testing"

	secondImageName, _, err := utils.GenerateImageNameFromString("docker.io/test2")
	require.NoError(t, err)
	e := &enricherImpl{
		imageGetter: func(ctx context.Context, id string) (*storage.Image, bool, error) {
			img.Signature = &storage.ImageSignature{Signatures: []*storage.Signature{createSignature("test", "test")}}
			img.SignatureVerificationData = &storage.ImageSignatureVerificationData{Results: []*storage.ImageSignatureVerificationResult{
				createSignatureVerificationResult("test", storage.ImageSignatureVerificationResult_VERIFIED)}}
			img.Names = append(img.Names, secondImageName)
			return img, true, nil
		},
	}
	imgFetchedFromDB, exists := e.fetchFromDatabase(context.Background(), img, UseImageNamesRefetchCachedValues)
	assert.False(t, exists)
	assert.Equal(t, img.GetName(), imgFetchedFromDB.GetName())
	assert.ElementsMatch(t, img.GetNames(), imgFetchedFromDB.GetNames())
	assert.Nil(t, img.GetSignature())
	assert.Nil(t, img.GetSignatureVerificationData())
}

func TestFetchFromDatabase_RetainImageNames(t *testing.T) {
	cimg, err := utils.GenerateImageFromString("docker.io/test")
	require.NoError(t, err)
	img := imgTypes.ToImage(cimg)
	img.Id = "sample-SHA"
	testImageName, _, err := utils.GenerateImageNameFromString(img.GetName().GetFullName())
	require.NoError(t, err)

	cimg, err = utils.GenerateImageFromString("docker.io/test2")
	require.NoError(t, err)
	existingImg := imgTypes.ToImage(cimg)
	existingImg.Id = "sample-SHA"
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
			testImg := img.Clone()
			_ = e.updateImageFromDatabase(context.Background(), testImg, testCase.opt)
			assert.ElementsMatch(t, testImg.GetNames(), testCase.expectedImageNames)
		})
	}
}
