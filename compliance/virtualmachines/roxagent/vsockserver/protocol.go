package vsockserver

import (
	"crypto/tls"
	"errors"
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
func (c *ReportCache) SetReport(r *v4.IndexReport, facts map[string]string) {
	var prev reportSnapshot
	if old := c.snap.Load(); old != nil {
		prev = *old
	}
	c.snap.Store(&reportSnapshot{
		report:      r,
		generation:  prev.generation + 1,
		generatedAt: time.Now(),
		facts:       facts,
	})
}

// Handler processes incoming VSOCK protocol requests.
type Handler struct {
	cache        *ReportCache
	agentVersion string
}

// NewHandler creates a protocol handler.
func NewHandler(cache *ReportCache, agentVersion string) *Handler {
	return &Handler{cache: cache, agentVersion: agentVersion}
}

// HandleConn reads a framed request from conn, processes it, writes a framed response, and closes conn.
func (h *Handler) HandleConn(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	reqData, err := vsockframing.ReadFrame(conn, maxRequestSize)
	if err != nil {
		if isTLSRecordError(err) {
			log.Warnf("Rejected plaintext connection from %s (peer not using TLS)", conn.RemoteAddr())
		} else {
			log.Errorf("Reading request frame: %v", err)
		}
		return
	}

	var req pb.VMServiceRequest
	if err := proto.Unmarshal(reqData, &req); err != nil {
		log.Errorf("Unmarshalling request: %v", err)
		h.writeError(conn, pb.ErrorCode_ERROR_CODE_INTERNAL, "malformed request")
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
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_NOT_READY, "initial scan in progress, try again later")
	}

	// Strict equality (not >=) so that after an agent restart — when the generation
	// counter resets to 1 — a sensor still holding a higher generation from the
	// previous instance will receive the full report instead of a false "unchanged".
	if req.GetIfNewerThanGeneration() == snap.generation {
		log.Infof("GetReport: unchanged (generation=%d, requested_if_newer=%d)", snap.generation, req.GetIfNewerThanGeneration())
		resp := h.newResponseFromSnap(snap)
		resp.Result = &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{Unchanged: true},
		}
		return resp
	}

	log.Infof("GetReport: serving report (generation=%d, packages=%d)", snap.generation, len(snap.report.GetContents().GetPackages()))
	resp := h.newResponseFromSnap(snap)
	resp.Result = &pb.VMServiceResponse_GetReport{
		GetReport: &pb.GetReportResponse{IndexReport: snap.report},
	}
	return resp
}

func (h *Handler) newResponse() *pb.VMServiceResponse {
	return h.newResponseFromSnap(h.cache.snap.Load())
}

func (h *Handler) newResponseFromSnap(snap *reportSnapshot) *pb.VMServiceResponse {
	facts := map[string]string{}
	var gen uint32
	if snap != nil {
		if snap.facts != nil {
			facts = snap.facts
		}
		gen = snap.generation
	}
	meta := &pb.ResponseMeta{
		AgentVersion:     h.agentVersion,
		ReportGeneration: gen,
		SupportedMethods: []string{"get_report"},
		Facts:            facts,
	}
	if snap != nil && !snap.generatedAt.IsZero() {
		meta.ReportGeneratedAt = timestamppb.New(snap.generatedAt)
	}
	return &pb.VMServiceResponse{Meta: meta}
}

func (h *Handler) errorResponse(code pb.ErrorCode, msg string) *pb.VMServiceResponse {
	resp := h.newResponse()
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
	var recordErr tls.RecordHeaderError
	return errors.As(err, &recordErr)
}
