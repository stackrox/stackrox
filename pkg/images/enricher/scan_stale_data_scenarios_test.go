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
	reporterMocks "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	"github.com/stackrox/rox/pkg/registries/types"
	registryMocks "github.com/stackrox/rox/pkg/registries/mocks"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannertypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// This file contains five scenario tests for the ROX-36267 failure family,
// where cached scan data does not reflect the current vulnerability definitions. Each test asserts
// the DESIRED behavior and therefore FAILS on master: they are the
// reproduction. When a fix lands, they go green and become the regression
// guard.

// scenarioScanner is an ImageScanner/Registry whose behavior can be switched
// mid-test to simulate vulnerability-store state transitions.
type scenarioScanner struct {
	mutex     sync.Mutex
	scan      *storage.ImageScan
	scanCalls int
	scanErr   error
	defsTs    *timestamppb.Timestamp
}

func (s *scenarioScanner) setScan(scan *storage.ImageScan) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scan = scan
}

func (s *scenarioScanner) setError(err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scanErr = err
}

func (s *scenarioScanner) setVulnDefinitions(ts *timestamppb.Timestamp) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.defsTs = ts
}

func (s *scenarioScanner) callCount() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.scanCalls
}

func (*scenarioScanner) Type() string                        { return "fake-v4" }
func (*scenarioScanner) Name() string                        { return "scenario-scanner" }
func (*scenarioScanner) Match(*storage.ImageName) bool       { return true }
func (*scenarioScanner) Test() error                         { return nil }
func (*scenarioScanner) Config(_ context.Context) *types.Config { return nil }
func (*scenarioScanner) HTTPClient() *http.Client             { return nil }
func (*scenarioScanner) ListTags(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (*scenarioScanner) Source() *storage.ImageIntegration {
	return &storage.ImageIntegration{Id: "id", Name: "scenario-scanner"}
}
func (*scenarioScanner) DataSource() *storage.DataSource {
	return &storage.DataSource{Id: "id", Name: "scenario-scanner"}
}
func (s *scenarioScanner) GetScanner() scannertypes.Scanner { return s }
func (*scenarioScanner) MaxConcurrentScanSemaphore() *semaphore.Weighted {
	return scenarioTestSemaphore
}
func (s *scenarioScanner) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.defsTs == nil {
		return &v1.VulnDefinitionsInfo{}, nil
	}
	return &v1.VulnDefinitionsInfo{LastUpdatedTimestamp: s.defsTs}, nil
}
func (s *scenarioScanner) GetScan(_ *storage.Image) (*storage.ImageScan, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.scanCalls++
	if s.scanErr != nil {
		return nil, s.scanErr
	}
	return s.scan, nil
}
func (s *scenarioScanner) Metadata(*storage.Image) (*storage.ImageMetadata, error) {
	return &storage.ImageMetadata{LayerShas: []string{"SHA1"}}, nil
}

var scenarioTestSemaphore = semaphore.NewWeighted(1)

// scanWithVulns returns a scan with a fixable CVE (CVSS 9.8).
func scanWithVulns(scanTime time.Time) *storage.ImageScan {
	return &storage.ImageScan{
		ScanTime: timestamppb.New(scanTime),
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

// scanWithoutVulns returns a scan with components but no vulnerabilities,
// simulating an image scanned while the vulnerability store was loading.
func scanWithoutVulns(scanTime time.Time) *storage.ImageScan {
	return &storage.ImageScan{
		ScanTime: timestamppb.New(scanTime),
		Components: []*storage.EmbeddedImageScanComponent{
			{Name: "alpine-baselayout", Version: "3.9.0"},
		},
	}
}

// setupScenarioEnricher creates an enricher with the given scanner and a
// stateful image store (stored scans are returned on subsequent lookups).
func setupScenarioEnricher(t *testing.T, scanner *scenarioScanner) (*enricherV2Impl, **storage.ImageV2) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)

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
	scannerSet.EXPECT().GetAll().Return([]scannertypes.ImageScannerWithDataSource{scanner}).AnyTimes()
	registrySet := registryMocks.NewMockSet(ctrl)
	registrySet.EXPECT().IsEmpty().Return(false).AnyTimes()
	registrySet.EXPECT().GetAllUnique().Return([]types.ImageRegistry{scanner}).AnyTimes()
	registrySet.EXPECT().Get(gomock.Any()).Return(scanner).AnyTimes()
	integrationSet := integrationMocks.NewMockSet(ctrl)
	integrationSet.EXPECT().ScannerSet().Return(scannerSet).AnyTimes()
	integrationSet.EXPECT().RegistrySet().Return(registrySet).AnyTimes()
	mockReporter := reporterMocks.NewMockReporter(ctrl)
	mockReporter.EXPECT().UpdateIntegrationHealthAsync(gomock.Any()).AnyTimes()

	enricher := &enricherV2Impl{
		integrationHealthReporter:  mockReporter,
		cvesSuppressor:             &fakeCVESuppressorV2{},
		integrations:               integrationSet,
		errorsPerScanner:           map[scannertypes.ImageScannerWithDataSource]int32{},
		errorsPerRegistry:          map[types.ImageRegistry]int32{},
		metadataLimiter:            rate.NewLimiter(rate.Every(50*time.Millisecond), 1),
		metadataCache:              newCache(),
		metrics:                    newMetrics(pkgMetrics.CentralSubsystem),
		imageGetter:                imageStore,
		baseImageGetter:            emptyBaseImageGetterV2,
		signatureIntegrationGetter: emptySignatureIntegrationGetter,
		signatureFetcher:           &fakeSigFetcher{},
	}
	return enricher, &stored
}

func newScenarioImage() *storage.ImageV2 {
	return &storage.ImageV2{
		Id:     imgUtils.NewImageV2ID(&storage.ImageName{Registry: "reg", FullName: "reg"}, "sha"),
		Digest: "sha",
		Name:   &storage.ImageName{Registry: "reg", FullName: "reg"},
		Metadata: &storage.ImageMetadata{
			LayerShas: []string{"SHA1"},
		},
	}
}

// Scenario 1: A scan stored during the vulnerability-store loading window is
// served by every subsequent force=false request, even after the data loads.
//
// This is the primary ROX-36267 failure: the matcher reports ready while its
// data is still loading, a scan succeeds with components but no scored
// vulnerabilities, Central stores it, and all later force=false requests
// (the default for /v1/images/scan and detectBuildTime) serve the stored
// scan.
func TestScenario1_InitialLoadWindow(t *testing.T) {
	scanner := &scenarioScanner{}
	now := time.Now()

	// The vulnerability store is loading: the scanner returns an indexed
	// image with components but no vulnerabilities.
	scanner.setScan(scanWithoutVulns(now.Add(-2 * time.Minute)))

	enricher, stored := setupScenarioEnricher(t, scanner)

	// Pre-scan during the loading window (what AdmissionControllerTest does).
	img1 := newScenarioImage()
	_, err := enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img1)
	require.NoError(t, err)
	require.NotNil(t, img1.GetScan())
	assert.NotEmpty(t, img1.GetScan().GetComponents(), "components present (indexed)")
	assert.Empty(t, img1.GetScan().GetComponents()[0].GetVulns(), "no vulnerabilities (store loading)")

	// Central stores the scan.
	*stored = img1

	// The vulnerability data finishes loading.
	scanner.setScan(scanWithVulns(now))

	// A force=false request (what the failing tests use): the stored
	// incomplete scan is served as current.
	img2 := newScenarioImage()
	_, err = enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img2)
	require.NoError(t, err)

	// DESIRED: after the vulnerability data loads, a force=false request
	// should reflect the loaded data.
	require.NotNil(t, img2.GetScan())
	assert.NotEmpty(t, img2.GetScan().GetComponents()[0].GetVulns(),
		"Scenario 1 (initial load window): force=false serves the stored "+
			"vulnerability-less scan after the data loaded")
}

// Scenario 2: After the defs timestamp advances (signaling "data updated"),
// the post-processing (GC, distManager) may still be running. A re-scan
// during this gap gets partial data. Since the new scan's timestamp is after
// the defs timestamp, it is served as current — the cache re-fills with
// incomplete data in a different way.
//
// This demonstrates that a defs-timestamp-based staleness check does not
// fully solve the problem: the timestamp advances before the data is
// completely settled.
func TestScenario2_DefsUpdateGap(t *testing.T) {
	scanner := &scenarioScanner{}
	now := time.Now()

	// Initial state: good data, scan stored.
	scanner.setScan(scanWithVulns(now.Add(-5 * time.Minute)))
	scanner.setVulnDefinitions(timestamppb.New(now.Add(-5 * time.Minute)))

	enricher, stored := setupScenarioEnricher(t, scanner)

	img1 := newScenarioImage()
	_, err := enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img1)
	require.NoError(t, err)
	*stored = img1

	// A defs update begins. The timestamp advances (all bundle rows set),
	// but the GC/distManager are still running — the scanner returns
	// partial data.
	defsUpdated := now.Add(-1 * time.Minute)
	scanner.setVulnDefinitions(timestamppb.New(defsUpdated))
	scanner.setScan(scanWithoutVulns(now)) // partial data during the gap

	// A forced re-scan (or a staleness check that triggers a re-scan) gets
	// the partial data and stores it.
	img2 := newScenarioImage()
	_, err = enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: ForceRefetch}, img2)
	require.NoError(t, err)
	*stored = img2

	// The gap ends; the scanner now has complete data.
	scanner.setScan(scanWithVulns(now))

	// A force=false request: the partial scan from the gap is served.
	// Its timestamp (now) is AFTER the defs timestamp (defsUpdated), so
	// even a correct staleness check would consider it current.
	img3 := newScenarioImage()
	_, err = enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img3)
	require.NoError(t, err)

	// DESIRED: after the defs-update gap ends and the data is complete,
	// a force=false request should reflect the complete data. The partial
	// scan from the gap has a timestamp after the defs update, so no
	// timestamp-based staleness check can detect it.
	require.NotNil(t, img3.GetScan())
	assert.NotEmpty(t, img3.GetScan().GetComponents()[0].GetVulns(),
		"Scenario 2 (defs-update gap): the partial scan from the gap is "+
			"served as current")
}

// Scenario 3: Central's scan cache has no invalidation mechanism tied to
// vulnerability-data changes. A scan stored at time T is served indefinitely,
// regardless of how many times the vulnerability definitions update. Only a
// forced re-scan or the 4-hour reprocessor can refresh it.
//
// This is the architectural root of Scenarios 1 and 2: without invalidation,
// any scan stored during a bad window persists until something external
// clears it.
func TestScenario3_NoCacheInvalidationOnDataChanges(t *testing.T) {
	scanner := &scenarioScanner{}
	now := time.Now()

	// Store a scan with specific vulnerabilities.
	originalVulns := scanWithVulns(now.Add(-8 * time.Hour)) // 8 hours ago
	scanner.setScan(originalVulns)

	enricher, stored := setupScenarioEnricher(t, scanner)

	img1 := newScenarioImage()
	_, err := enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img1)
	require.NoError(t, err)
	*stored = img1

	// The vulnerability data changes COMPLETELY: different CVEs, different
	// CVSS scores, different fix statuses. The defs have updated many times.
	scanner.setScan(scanWithoutVulns(now)) // completely different data
	scanner.setVulnDefinitions(timestamppb.New(now))

	// A force=false request 8 hours later: the original scan is still served.
	// No amount of vulnerability-data change invalidates the cache.
	img2 := newScenarioImage()
	_, err = enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img2)
	require.NoError(t, err)

	// DESIRED: a force=false request should reflect the current
	// vulnerability data, not data from 8 hours ago. The scan cache has
	// no invalidation mechanism tied to vulnerability-data changes.
	require.NotNil(t, img2.GetScan())
	assert.Empty(t, img2.GetScan().GetComponents()[0].GetVulns(),
		"Scenario 3 (no cache invalidation): the 8-hour-old scan with "+
			"obsolete CVEs is served as current; the cache is never "+
			"invalidated when vulnerability data changes")
}

// Scenario 4: When the scan pipeline fails (scanner unavailable, matcher not
// ready, registry error), a force=false request still returns the stored
// scan as a SUCCESS. The error is invisible to the consumer. This is why
// the CI failures looked like "missing data" rather than "scanner broken."
func TestScenario4_ErrorMaskingAsStaleSuccess(t *testing.T) {
	scanner := &scenarioScanner{}
	now := time.Now()

	// Store a good scan.
	scanner.setScan(scanWithVulns(now.Add(-10 * time.Minute)))

	enricher, stored := setupScenarioEnricher(t, scanner)

	img1 := newScenarioImage()
	_, err := enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img1)
	require.NoError(t, err)
	*stored = img1

	// The scan pipeline breaks completely: the scanner starts failing.
	scanner.setError(assert.AnError)

	// A force=false request: the stored scan is served as a SUCCESS.
	// The scanner is never called (useExistingScan short-circuits), so the
	// error is invisible. The consumer sees clean, current-looking data.
	img2 := newScenarioImage()
	result, err := enricher.EnrichImage(context.Background(),
		EnrichmentContext{FetchOpt: UseCachesIfPossible}, img2)

	// DESIRED: when the scan pipeline is broken, the consumer should
	// see an error or a clear indication that the data is stale. Serving
	// cached data as a success with no warning makes pipeline failures
	// look like missing data.
	require.NotNil(t, img2.GetScan())
	assert.NotEqual(t, ScanReused, result.ScanResult,
		"Scenario 4 (error masking): the stored scan is served as a "+
			"success with no indication that the pipeline is broken; "+
			"consumers cannot distinguish stale from current data")
}

// Scenario 5: The QA test framework's retry pattern (evaluateWithRetry)
// retries errors only. A successful-but-incomplete scan is final — no
// retry, no re-check. The pre-scan pattern (deliberately populating the
// cache before assertions) locks in whatever the first scan returned.
// Subsequent "retries" serve the same cached incomplete data.
//
// This is why the CI tests could not self-heal: the first scan during a
// bad window poisons the entire test run, and the retry loop exits
// successfully with the bad data.
func TestScenario5_RetryLoopLocksInBadResults(t *testing.T) {
	scanner := &scenarioScanner{}
	now := time.Now()

	// The vulnerability store is loading.
	scanner.setScan(scanWithoutVulns(now))

	enricher, stored := setupScenarioEnricher(t, scanner)

	// Simulate the test framework's retry pattern: call with force=false,
	// retry only on error. This is what evaluateWithRetry does.
	attempt := func() (*storage.ImageV2, error) {
		img := newScenarioImage()
		_, err := enricher.EnrichImage(context.Background(),
			EnrichmentContext{FetchOpt: UseCachesIfPossible}, img)
		return img, err
	}

	// First attempt: succeeds with incomplete data. No error → retry loop
	// would exit here in the real test framework.
	img1, err := attempt()
	require.NoError(t, err, "first attempt succeeds (no error to retry)")
	*stored = img1 // the pre-scan populates the cache

	// The vulnerability data finishes loading.
	scanner.setScan(scanWithVulns(now))

	// DESIRED: after the vulnerability data loads, a retry should
	// eventually return the loaded data. The retry pattern (retry on
	// error only) cannot self-heal this failure mode because the
	// response is always a success with bad data.
	for i := 0; i < 5; i++ {
		img, err := attempt()
		require.NoError(t, err, "retry %d", i)
		require.NotNil(t, img.GetScan())
		assert.NotEmpty(t, img.GetScan().GetComponents()[0].GetVulns(),
			"Scenario 5 (retry locks in): retry %d still serves the "+
				"cached incomplete data", i)
		if len(img.GetScan().GetComponents()[0].GetVulns()) > 0 {
			break // data recovered, test passes
		}
	}
}
