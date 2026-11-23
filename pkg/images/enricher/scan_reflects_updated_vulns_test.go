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
	imgUtils "github.com/stackrox/rox/pkg/images/utils"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	registriesTypes "github.com/stackrox/rox/pkg/registries/types"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// TestScanReflectsUpdatedVulnerabilityData reproduces the ROX-36267 failure:
// a scan stored while the scanner's vulnerability data was incomplete is
// served by every subsequent force=false enrichment - even after the scanner
// would return complete data - so vulnerability policies silently do not fire.
//
// This test asserts the DESIRED behavior and therefore FAILS on master; it is
// the reproduction. A fix (e.g. invalidating stored scans when vulnerability
// definitions update, or refusing to cache scans from an incomplete store)
// makes it pass. See the accompanying characterization test,
// TestStoredScanServedAfterScannerDataChanges, which documents the current
// behavior step by step.
//
// Why this matters (the CI failure this produced): the image is indexed
// (components present) during the vulnerability-store loading window, the
// scan succeeds with zero scored vulnerabilities, Central stores it, and all
// later force=false scans - the default for /v1/images/scan and
// detectBuildTime, i.e. exactly what the QA tests use - serve the stored
// empty scan. The response is a success, so retry loops do not help. The
// tests see components-but-no-vulnerabilities, which is indistinguishable
// from a legitimately clean image.

// switchableScanner is an ImageScanner/Registry whose scan result can be
// switched mid-test, simulating the vulnerability store loading: first the
// incomplete window (components but no vulnerabilities), then the loaded
// state (components with fixable, scored vulnerabilities).
type switchableScanner struct {
	mutex     sync.Mutex
	scan      *storage.ImageScan
	scanCalls int
}

func (s *switchableScanner) setScan(scan *storage.ImageScan) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scan = scan
}

func (s *switchableScanner) callCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.scanCalls
}

func (*switchableScanner) Type() string                        { return "fake-v4" }
func (*switchableScanner) Name() string                        { return "fake-scanner-v4" }
func (*switchableScanner) Match(*storage.ImageName) bool        { return true }
func (*switchableScanner) Test() error                         { return nil }
func (*switchableScanner) Config(_ context.Context) *registriesTypes.Config { return nil }
func (*switchableScanner) HTTPClient() *http.Client            { return nil }
func (*switchableScanner) ListTags(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (*switchableScanner) Source() *storage.ImageIntegration {
	return &storage.ImageIntegration{Id: "id", Name: "fake-scanner-v4"}
}
func (*switchableScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{Id: "id", Name: "fake-scanner-v4"}
}
func (s *switchableScanner) GetScanner() types.Scanner { return s }

var scanTestSemaphore = semaphore.NewWeighted(1)

func (*switchableScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return scanTestSemaphore
}

func (s *switchableScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return &v1.VulnDefinitionsInfo{}, nil
}

func (s *switchableScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scanCalls++
	return s.scan, nil
}

func (s *switchableScanner) Metadata(*storage.Image) (*storage.ImageMetadata, error) {
	return &storage.ImageMetadata{LayerShas: []string{"SHA1"}}, nil
}

// scanWithoutVulns models the matcher state during the load window:
// the image is indexed (components present) but vulnerability enrichment
// has not completed, so there are no scored, fixable vulnerabilities.
func scanWithoutVulns() *storage.ImageScan {
	return &storage.ImageScan{
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "alpine-baselayout", Version: "3.9.0"},
		},
	}
}

// scanWithVulns models the state after the vulnerability data loaded.
func scanWithVulns() *storage.ImageScan {
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

func TestScanReflectsUpdatedVulnerabilityData(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)

	scanner := &switchableScanner{}
	scanner.setScan(scanWithoutVulns())

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
		integrationHealthReporter:  mockReporter,
		cvesSuppressor:             &fakeCVESuppressorV2{},
		integrations:               integrationSet,
		errorsPerScanner:           map[types.ImageScannerWithDataSource]int32{},
		errorsPerRegistry:          map[registriesTypes.ImageRegistry]int32{},
		metadataLimiter:            rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:              newCache(),
		metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
		imageGetter:                imageStore,
		baseImageGetter:            emptyBaseImageGetterV2,
		signatureIntegrationGetter: emptySignatureIntegrationGetter,
		signatureFetcher:           &fakeSigFetcher{},
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

	// Phase 1: a scan during the vulnerability-store loading window. The
	// scanner returns an indexed image with no scored vulnerabilities;
	// Central stores this scan.
	img1 := newImage()
	result, err := enricherImpl.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img1)
	require.NoError(t, err)
	assert.Equal(t, ScanSucceeded, result.ScanResult)
	require.NotNil(t, img1.GetScan())
	assert.NotEmpty(t, img1.GetScan().GetComponents(),
		"components are present (the image was indexed)")
	assert.Empty(t, img1.GetScan().GetComponents()[0].GetVulns(),
		"no vulnerabilities yet (vulnerability store still loading)")

	stored = img1 // Central saves the enriched image.

	// Phase 2: the vulnerability data finished loading. A force=false
	// enrichment (the default for /v1/images/scan and detectBuildTime) must
	// NOT keep serving the stored vulnerability-less scan as though it were
	// current: the scanner's data has changed, and serving the stale scan
	// makes vulnerability policies silently not fire.
	scanner.setScan(scanWithVulns())

	img2 := newImage()
	_, err = enricherImpl.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img2)
	require.NoError(t, err)
	require.NotNil(t, img2.GetScan())

	// THE REPRODUCTION: this assertion fails on master. The stored scan from
	// the loading window is served as current even though the scanner now
	// returns complete data. Fix this (invalidate or refresh stored scans
	// when vulnerability definitions change) and the test goes green.
	assert.NotEmpty(t, img2.GetScan().GetComponents()[0].GetVulns(),
		"REPRODUCTION (ROX-36267): a force=false scan after the vulnerability "+
			"data loaded still serves the stored vulnerability-less scan; "+
			"vulnerability policies silently do not fire for this image")
}
