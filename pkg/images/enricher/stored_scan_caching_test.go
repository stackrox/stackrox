package enricher

import (
	"context"
	"net/http"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	integrationMocks "github.com/stackrox/rox/pkg/images/integration/mocks"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	registriesTypes "github.com/stackrox/rox/pkg/registries/types"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// PROBLEM: vulnerability policies can silently not fire for images whose
// scan was stored during a vulnerability-data loading window.
//
// MECHANISM (demonstrated by the test below):
//
//	1. The scanner (e.g. Scanner V4 matcher) reports ready before its
//	   vulnerability data is fully loaded (see ROX-36267, ROX-36688).
//	2. A scan of an image during that window succeeds: the image is
//	   indexed (components present) but has no scored vulnerabilities.
//	   Central stores this scan.
//	3. Every subsequent scan request with force=false (the default for
//	   both /v1/images/scan and detectBuildTime) reuses the STORED scan
//	   without consulting the scanner - even after the vulnerability
//	   data has finished loading. The stored scan is indistinguishable
//	   from a legitimately clean scan.
//	4. Vulnerability policies (e.g. "Fixable CVSS >= 7") therefore do not
//	   fire for that image, silently, for as long as the stored scan is
//	   served.
//
// The CI test failures this family produced were confounding for years
// precisely because of step 3: the data was loaded and the matcher was
// serving correct results, but the tests read the cached empty scan.
// Retry loops did not help because the response is a success, not an error.
//
// FIX DIRECTION: invalidate (or refuse to serve) stored scans whose
// vulnerability data predates the latest vulnerability-definition update.
// When such a fix lands, the Phase 2 assertions below should be flipped to
// expect the fresh scan; this test then becomes the regression guard.
//
// See also the e2e scenarios in scripts/ci/cvss-failure-modes for the
// end-to-end demonstration (trigger + amplifier + observed failure).

// switchableFakeScanner is an ImageScanner whose scan result can be switched
// mid-test, simulating the vulnerability store loading: first the incomplete
// window (components but no vulnerabilities), then the loaded state
// (components with fixable, scored vulnerabilities).
type switchableFakeScanner struct {
	mutex     sync.Mutex
	scan      *storage.ImageScan
	scanCalls int
}

func (s *switchableFakeScanner) setScan(scan *storage.ImageScan) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scan = scan
}

func (s *switchableFakeScanner) callCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.scanCalls
}

func (*switchableFakeScanner) Type() string  { return "fake-v4" }
func (*switchableFakeScanner) Name() string  { return "fake-scanner-v4" }
func (*switchableFakeScanner) Match(*storage.ImageName) bool { return true }
func (*switchableFakeScanner) Test() error                   { return nil }

// Registry interface (modeled on fakeRegistryScanner in testing_test.go):
// the switchable scanner doubles as the image registry so the enrichment
// flow can fetch metadata.
func (s *switchableFakeScanner) Metadata(*storage.Image) (*storage.ImageMetadata, error) {
	return &storage.ImageMetadata{LayerShas: []string{"SHA1"}}, nil
}
func (*switchableFakeScanner) Config(_ context.Context) *registriesTypes.Config { return nil }
func (*switchableFakeScanner) HTTPClient() *http.Client                        { return nil }
func (*switchableFakeScanner) ListTags(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (*switchableFakeScanner) Source() *storage.ImageIntegration {
	return &storage.ImageIntegration{Id: "id", Name: "fake-scanner-v4"}
}

func (*switchableFakeScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{Id: "id", Name: "fake-scanner-v4"}
}

// GetScanner returns the scanner itself (it implements the scan interface).
func (s *switchableFakeScanner) GetScanner() types.Scanner {
	return s
}

// MaxConcurrentScanSemaphore for the scan flow.
var scanSemaphore = semaphore.NewWeighted(1)

func (*switchableFakeScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return scanSemaphore
}

func (s *switchableFakeScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

func (s *switchableFakeScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scanCalls++
	return s.scan, nil
}

// incompleteWindowScan models the matcher state during the load window:
// the image is indexed (components present) but vulnerability enrichment
// has not completed, so there are no (fixable, scored) vulnerabilities.
func incompleteWindowScan() *storage.ImageScan {
	return &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "alpine-baselayout", Version: "3.9.0"},
		},
	}
}

// loadedScan models the state after the vulnerability data finished loading.
func loadedScan() *storage.ImageScan {
	return &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{
				Name:    "alpine-baselayout",
				Version: "3.9.0",
				FixedBy: "3.9.1",
				Vulns: []*storage.EmbeddedVulnerability{
					{Cve: "CVE-2020-1234", Cvss: 9.8},
				},
			},
		},
	}
}

func TestStoredScanServedAfterScannerDataChanges(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)

	scanner := &switchableFakeScanner{}
	scanner.setScan(incompleteWindowScan())

	// imageStore simulates Central's image datastore: whatever the enrichment
	// produced gets saved and is returned for subsequent enrichments.
	var stored *storage.ImageV2
	imageStore := func(_ context.Context, _ string) (*storage.ImageV2, bool, error) {
		if stored == nil {
			return nil, false, nil
		}
		return stored, true, nil
	}

	ctrl := gomock.NewController(t)
	scannerSet := scannerMocks.NewMockSet(ctrl)
	scannerSet.EXPECT().IsEmpty().Return(false).AnyTimes()
	scannerSet.EXPECT().GetAll().Return([]types.ImageScannerWithDataSource{scanner}).AnyTimes()
	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySet.EXPECT().GetAllUnique().Return([]registriesTypes.ImageRegistry{scanner}).AnyTimes()
	registrySet.EXPECT().Get(gomock.Any()).Return(scanner).AnyTimes()
	integrationSet := integrationMocks.NewMockSet(ctrl)
	integrationSet.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()
	integrationSet.EXPECT().RegistrySet().Return(registrySet).AnyTimes()

	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricherImpl := &enricherV2Impl{
		integrationHealthReporter: mockReporter,
		cvesSuppressor:            &fakeCVESuppressorV2{},
		integrations:      integrationSet,
		errorsPerScanner:  map[types.ImageScannerWithDataSource]int32{},
		errorsPerRegistry: map[registriesTypes.ImageRegistry]int32{},
		metadataLimiter:   rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:     newCache(),
		metrics:           newMetrics(pkgMetrics.CentralSubsystem),
		imageGetter:       imageStore,
		baseImageGetter:             emptyBaseImageGetterV2,
		signatureIntegrationGetter:  emptySignatureIntegrationGetter,
		signatureFetcher:            &fakeSigFetcher{},
	}

	newImage := func() *storage.ImageV2 {
		return &storage.ImageV2{
			Id:     imgUtils.NewImageV2ID(&storage.ImageName{Registry: "reg", FullName: "reg"}, "sha"),
			Digest: "sha",
			Name:   &storage.ImageName{Registry: "reg", FullName: "reg"},
			Metadata: &storage.ImageMetadata{
				LayerShas: []string{"SHA1"},
			},
		}
	}

	// Phase 1: the "pre-scan" during the incomplete window. The scanner is
	// consulted and returns the incomplete scan; the image now carries it.
	// (In the CI failure this is AdmissionControllerTest's pre-scan, whose
	// comment literally says it exists to populate this cache.)
	img1 := newImage()
	result, err := enricherImpl.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img1)
	require.NoError(t, err)
	assert.Equal(t, ScanSucceeded, result.ScanResult)
	require.NotNil(t, img1.GetScan())
	assert.NotEmpty(t, img1.GetScan().GetComponents(), "components present (indexed)")
	assert.Empty(t, img1.GetScan().GetComponents()[0].GetVulns(), "no vulnerabilities yet (incomplete window)")

	// Central stores the enriched image.
	stored = img1

	// Phase 2: THE BUG. The vulnerability data finished loading - the scanner
	// would now return complete results - but a force=false enrichment
	// (the default for /v1/images/scan and detectBuildTime, i.e. exactly
	// what the CI tests use) serves the STORED incomplete scan instead.
	// The scanner is never consulted; the response is a success; callers
	// cannot distinguish this from a legitimately clean scan.
	scanner.setScan(loadedScan())
	callsBefore := scanner.callCount()

	img2 := newImage()
	result2, err := enricherImpl.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img2)
	require.NoError(t, err)

	// If these assertions FAIL, the caching was fixed: force=false now
	// reflects updated scanner data. Flip them to expect the good scan
	// (ScanSucceeded, scanner consulted, vulnerabilities present) and keep
	// this test as the regression guard.
	assert.Equal(t, ScanReused, result2.ScanResult,
		"BUG (documented): the stored incomplete scan is reused")
	assert.Equal(t, callsBefore, scanner.callCount(),
		"BUG (documented): the scanner was NOT consulted despite its data having changed")
	require.NotNil(t, img2.GetScan())
	assert.Empty(t, img2.GetScan().GetComponents()[0].GetVulns(),
		"BUG (documented): the served scan still has no vulnerabilities")

	// Phase 3: only a forced refetch replaces the stored scan.
	img3 := newImage()
	result3, err := enricherImpl.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: ForceRefetch}, img3)
	require.NoError(t, err)
	assert.Equal(t, ScanSucceeded, result3.ScanResult)
	assert.Equal(t, callsBefore+1, scanner.callCount(), "the forced refetch consulted the scanner")
	require.NotNil(t, img3.GetScan())
	assert.NotEmpty(t, img3.GetScan().GetComponents()[0].GetVulns(),
		"the forced refetch returned the loaded vulnerability data")
}
