package vsockserver

import (
	"maps"
	"sync/atomic"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"google.golang.org/protobuf/proto"
)

// ReportProvider supplies the report data Handler.handleGetReport serves,
// decoupling wire-protocol handling (protocol.go) from where a report
// actually comes from. It has two implementations: CacheReportProvider,
// backed by a real completed host scan, and FakeReportProvider (see
// loadtest_report_provider.go), which serves canned data for load testing.
//
// req and meta are passed through so a load-test implementation can vary
// its response by request (e.g. requested report size) without changing
// the wire protocol.
//
// generatedAt is returned alongside generation/epoch (rather than folded
// into report itself) because it is response metadata, not report content —
// mirrors ResponseMeta's own shape, where report_generated_at is a sibling
// field to the report, not part of it.
type ReportProvider interface {
	GetReport(req *pb.GetReportRequest, meta *pb.RequestMeta) (report *v4.IndexReport, facts map[string]string, generation uint32, epoch uint32, generatedAt time.Time, ready bool)
}

// reportSnapshot is an immutable point-in-time view of the cached report state.
type reportSnapshot struct {
	report      *v4.IndexReport
	generation  uint32
	generatedAt time.Time
	facts       map[string]string
}

// ReportCache holds the cached scan report with its generation counter.
// Invariant: exactly one goroutine (the rescan loop) calls SetReport; multiple
// HandleConn goroutines read via snap.Load(). This single-writer/multi-reader
// pattern is safe with atomic.Pointer without CAS.
type ReportCache struct {
	snap atomic.Pointer[reportSnapshot]
}

// SetReport atomically publishes a new report with updated facts in a single
// store, incrementing the generation counter. Readers never observe a partial
// (new report, stale facts) state.
//
// r and facts are defensively copied so that a caller mutating its own copy
// after this call (or reusing the same facts map across scans) can never
// mutate the published, supposedly-immutable snapshot out from under
// concurrent readers.
func (c *ReportCache) SetReport(r *v4.IndexReport, facts map[string]string) {
	var prev reportSnapshot
	if old := c.snap.Load(); old != nil {
		prev = *old
	}
	c.snap.Store(&reportSnapshot{
		report:      cloneIndexReport(r),
		generation:  prev.generation + 1,
		generatedAt: time.Now(),
		facts:       cloneFacts(facts),
	})
}

// cloneIndexReport returns a deep copy of r, or nil if r is nil.
func cloneIndexReport(r *v4.IndexReport) *v4.IndexReport {
	if r == nil {
		return nil
	}
	return proto.Clone(r).(*v4.IndexReport)
}

// cloneFacts returns a shallow copy of in, or nil if in is empty.
func cloneFacts(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	maps.Copy(out, in)
	return out
}

// CacheReportProvider is the production ReportProvider: a real host scan
// produces exactly one report at a time, so req/meta (which only matter for
// a load-test provider varying its response per request) are irrelevant here
// and ignored.
type CacheReportProvider struct {
	cache *ReportCache
	// epoch is seeded once per process lifetime and never persisted to VM
	// disk. It lets Sensor distinguish "this agent restarted" from "this
	// agent's generation counter coincidentally reset to a value Sensor
	// already has cached" (see ROX-35597) without changing report_generation's
	// own sequential, human-readable semantics.
	epoch uint32
}

// NewCacheReportProvider wraps cache as a ReportProvider for NewHandler.
func NewCacheReportProvider(cache *ReportCache) *CacheReportProvider {
	return &CacheReportProvider{cache: cache, epoch: newEpoch()}
}

// GetReport implements ReportProvider.
//
// facts is cloned again on the way out (on top of SetReport's clone-on-write)
// so a caller mutating the returned map can never corrupt the cached
// snapshot that concurrent readers still see.
func (p *CacheReportProvider) GetReport(_ *pb.GetReportRequest, _ *pb.RequestMeta) (*v4.IndexReport, map[string]string, uint32, uint32, time.Time, bool) {
	snap := p.cache.snap.Load()
	if snap == nil || snap.report == nil {
		return nil, nil, 0, p.epoch, time.Time{}, false
	}
	return snap.report, cloneFacts(snap.facts), snap.generation, p.epoch, snap.generatedAt, true
}

// newEpoch derives a process-lifetime epoch value. Time-derived rather than
// cryptographically random: epoch only needs to differ across restarts with
// overwhelming probability, not resist an adversary. Seconds (not
// nanoseconds) since the Unix epoch, so the value only wraps every ~136
// years instead of every ~4.3 seconds when truncated to uint32. 0 is
// reserved to mean "agent predates this field" (see ResponseMeta.epoch doc),
// so it's excluded.
func newEpoch() uint32 {
	if e := uint32(time.Now().Unix()); e != 0 {
		return e
	}
	return 1
}
