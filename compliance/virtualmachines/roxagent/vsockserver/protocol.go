package vsockserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	"net"
	"sync/atomic"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxRequestSize = 1 << 20 // 1 MiB

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
	var counter uint32
	if old := c.snap.Load(); old != nil {
		counter = old.generation
	}
	c.snap.Store(&reportSnapshot{
		report:      cloneIndexReport(r),
		generation:  counter + 1,
		generatedAt: time.Now(),
		facts:       cloneFacts(facts),
	})
}

// cloneIndexReport returns a deep copy of r, or nil if r is nil.
func cloneIndexReport(r *v4.IndexReport) *v4.IndexReport {
	if r == nil {
		return nil
	}
	// proto.Clone always preserves r's concrete type, so this assertion cannot fail.
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

// Handler processes incoming VSOCK protocol requests.
type Handler struct {
	cache        *ReportCache
	agentVersion string
	// epoch is seeded once per process lifetime and never persisted to VM
	// disk. It lets Sensor (and this handler itself, via GetReportRequest's
	// known_epoch) distinguish "this agent restarted" from "this agent's
	// generation counter coincidentally reset to a value Sensor already has
	// cached" without changing report_generation's own sequential,
	// human-readable semantics.
	epoch uint32
}

// NewHandler creates a protocol handler.
func NewHandler(cache *ReportCache, agentVersion string) *Handler {
	return &Handler{cache: cache, agentVersion: agentVersion, epoch: newEpoch()}
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

// HandleConn reads a framed request from conn, processes it, writes a framed response, and closes conn.
func (h *Handler) HandleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reqData, err := vsockframing.ReadFrame(conn, maxRequestSize)
	if err != nil {
		switch {
		case isTLSRecordError(err):
			log.Warnf("Rejected plaintext connection from %s (peer not using TLS)", conn.RemoteAddr())
		case errors.Is(err, vsockframing.ErrFrameTooLarge):
			// The oversized length prefix was already read; the payload itself
			// was not, so the connection is still in a writable state.
			log.Warnf("Rejecting oversized request from %s: %v", conn.RemoteAddr(), err)
			h.writeError(conn, pb.ErrorCode_ERROR_CODE_REQUEST_TOO_LARGE, err.Error())
		default:
			log.Errorf("Reading request frame: %v", err)
		}
		return
	}

	var req pb.VMServiceRequest
	if err := proto.Unmarshal(reqData, &req); err != nil {
		log.Errorf("Unmarshalling request: %v", err)
		h.writeError(conn, pb.ErrorCode_ERROR_CODE_MALFORMED_REQUEST, fmt.Sprintf("malformed request: %v", err))
		return
	}

	resp := h.dispatch(&req)
	respData, err := proto.Marshal(resp)
	if err != nil {
		log.Errorf("Marshalling response: %v", err)
		return
	}
	if err := vsockframing.WriteFrame(conn, respData); err != nil {
		log.Errorf("Writing response frame: %v", err)
	}
}

func (h *Handler) dispatch(req *pb.VMServiceRequest) *pb.VMServiceResponse {
	switch req.GetMethod().(type) {
	case *pb.VMServiceRequest_GetReport:
		return h.handleGetReport(req.GetGetReport())
	default:
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD, "unknown or unset method")
	}
}

func (h *Handler) handleGetReport(req *pb.GetReportRequest) *pb.VMServiceResponse {
	snap := h.cache.snap.Load()
	if snap == nil || snap.report == nil {
		log.Info("GetReport: not ready (initial scan in progress)")
		return h.errorResponseFromSnap(snap, pb.ErrorCode_ERROR_CODE_NOT_READY, "initial scan in progress, try again later")
	}

	// Strict equality (not >=) so that after an agent restart — when the generation
	// counter resets to 1 — a sensor still holding a higher generation from the
	// previous instance will receive the full report instead of a false "unchanged".
	//
	// known_epoch guards against the opposite false positive: a restarted agent
	// whose reset generation coincidentally re-matches last_known_generation.
	// 0 means Sensor doesn't know our epoch yet (first-ever request for this VM,
	// or a Sensor build that predates the field), so fall back to generation-only
	// comparison exactly as before this field existed.
	generationMatches := req.GetLastKnownGeneration() == snap.generation
	knownEpoch := req.GetKnownEpoch()
	epochMatches := knownEpoch == 0 || knownEpoch == h.epoch
	if generationMatches && epochMatches {
		log.Infof("GetReport: unchanged (generation=%d, last_known_generation=%d)", snap.generation, req.GetLastKnownGeneration())
		resp := h.newResponseFromSnap(snap)
		resp.Result = &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{Unchanged: true},
		}
		return resp
	}
	if generationMatches && !epochMatches {
		log.Infof("GetReport: generation matches (%d) but epoch changed (known=%d, current=%d) — agent restarted, serving full report in this round trip",
			snap.generation, knownEpoch, h.epoch)
	}

	log.Infof("GetReport: serving report (generation=%d, packages=%d)", snap.generation, len(snap.report.GetContents().GetPackages()))
	resp := h.newResponseFromSnap(snap)
	resp.Result = &pb.VMServiceResponse_GetReport{
		GetReport: &pb.GetReportResponse{IndexReport: snap.report},
	}
	return resp
}

func (h *Handler) newResponseFromSnap(snap *reportSnapshot) *pb.VMServiceResponse {
	var facts map[string]string
	var gen uint32
	if snap != nil {
		facts = cloneFacts(snap.facts)
		gen = snap.generation
	}
	meta := &pb.ResponseMeta{
		AgentVersion:     h.agentVersion,
		ReportGeneration: gen,
		SupportedMethods: []string{"get_report"},
		Facts:            facts,
		Epoch:            h.epoch,
	}
	if snap != nil && !snap.generatedAt.IsZero() {
		meta.ReportGeneratedAt = timestamppb.New(snap.generatedAt)
	}
	return &pb.VMServiceResponse{Meta: meta}
}

func (h *Handler) errorResponse(code pb.ErrorCode, msg string) *pb.VMServiceResponse {
	return h.errorResponseFromSnap(h.cache.snap.Load(), code, msg)
}

// errorResponseFromSnap builds an error response using an already-loaded
// snapshot, so callers that loaded snap to make a decision (e.g.
// handleGetReport's NOT_READY check) don't race a concurrent SetReport
// between that load and the one newResponse would otherwise perform again.
func (h *Handler) errorResponseFromSnap(snap *reportSnapshot, code pb.ErrorCode, msg string) *pb.VMServiceResponse {
	resp := h.newResponseFromSnap(snap)
	resp.Result = &pb.VMServiceResponse_Error{
		Error: &pb.ErrorResponse{Code: code, Message: msg},
	}
	return resp
}

func (h *Handler) writeError(conn net.Conn, code pb.ErrorCode, msg string) {
	resp := h.errorResponse(code, msg)
	data, err := proto.Marshal(resp)
	if err != nil {
		return
	}
	_ = vsockframing.WriteFrame(conn, data)
}

func isTLSRecordError(err error) bool {
	_, ok := errors.AsType[tls.RecordHeaderError](err)
	return ok
}
