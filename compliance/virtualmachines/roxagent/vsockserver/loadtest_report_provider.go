package vsockserver

import (
	"math/rand/v2"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
)

// LOAD-TEST ONLY. Replaces a real host scan with canned reports so a fake
// roxagent can be deployed onto a real KubeVirt VM without any real
// NodeIndexer scanning. Only ever wired into NewHandler when
// fakeReportProviderEnabled is true (see cmd/serve.go).

const (
	reportSizeSmall  = "small"
	reportSizeMedium = "medium" // default, used when a request's report_size fact is absent or unrecognized.
	reportSizeLarge  = "large"

	// fakeReportProviderSeed is fixed (not time-derived) so canned report
	// content is identical across agent restarts -- only epoch should churn.
	fakeReportProviderSeed = 42
)

// fakeReportPackageCounts matches Part B's benchmark shapes
// (handler_bench_test.go, on stacked branch piotr/ROX-35195-vsock-roxagent-bench)
// for direct comparability against those per-size numbers.
var fakeReportPackageCounts = map[string]int{
	reportSizeSmall:  50,
	reportSizeMedium: 524,
	reportSizeLarge:  2000,
}

// FakeReportProvider is the load-test ReportProvider: it serves canned
// reports instead of a real host scan. One canned *v4.IndexReport per size
// preset is pre-built once at construction time; GetReport picks one
// per-request based on meta.Facts["report_size"] (defaulting to "medium" if
// absent/unrecognized) -- no redeploy needed to change size mid-experiment.
//
// Unlike CacheReportProvider, epoch is freshly randomized on every response
// rather than seeded once per process lifetime -- deliberately far more
// aggressive churn than a real roxagent would ever produce, forcing every
// hammer-mode request down the full-report path via Sensor's real, always-on
// epoch-mismatch dedup logic (ROX-35597), with no Sensor-side special-casing
// required. report_generation itself is NOT touched by this: each canned
// report has a fixed generation of 1 (built once, same as a real ReportCache
// after a single SetReport call) -- only epoch is load-test-special-cased.
type FakeReportProvider struct {
	reports     map[string]*v4.IndexReport
	facts       map[string]string
	generatedAt time.Time
}

// NewFakeReportProvider pre-builds the small/medium/large canned reports
// once. No periodic rescans, no host discovery.
func NewFakeReportProvider() *FakeReportProvider {
	reports := make(map[string]*v4.IndexReport, len(fakeReportPackageCounts))
	for size, numPackages := range fakeReportPackageCounts {
		reports[size] = vmindexreport.NewGeneratorWithSeed(numPackages, fakeReportProviderSeed).GenerateV4IndexReport()
	}
	return &FakeReportProvider{
		reports:     reports,
		facts:       map[string]string{"detected_os": "RHEL", "os_version": "9.4"},
		generatedAt: time.Now(),
	}
}

// GetReport implements ReportProvider.
func (p *FakeReportProvider) GetReport(_ *pb.GetReportRequest, meta *pb.RequestMeta) (*v4.IndexReport, map[string]string, uint32, uint32, time.Time, bool) {
	report, ok := p.reports[meta.GetFacts()["report_size"]]
	if !ok {
		report = p.reports[reportSizeMedium]
	}
	return report, p.facts, 1, fakeEpoch(), p.generatedAt, true
}

// fakeEpoch returns a freshly randomized, non-zero epoch on every call. 0 is
// reserved to mean "agent predates this field" (see ResponseMeta.epoch doc),
// so it's excluded, same as newEpoch().
func fakeEpoch() uint32 {
	if e := rand.Uint32(); e != 0 {
		return e
	}
	return 1
}
