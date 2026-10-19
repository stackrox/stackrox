package vsockserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"maps"
	"net"
	"slices"
	"sync/atomic"
	"time"

	"github.com/cespare/xxhash/v2"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stackrox/rox/pkg/vsockframing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// maxRequestSize must fit the largest message - a synced repo-to-CPE mapping (up to MaxMappingBytes) - plus envelope headroom.
	maxRequestSize = cpemapping.MaxMappingBytes + 1<<20

	methodGetReport          = "get_report"
	methodSyncRepoCPEMapping = "sync_repo_cpe_mapping"
)

// reportSnapshot is an immutable point-in-time view of the cached report state.
type reportSnapshot struct {
	report      *v4.IndexReport
	token       string
	generatedAt time.Time
	facts       map[string]string
	mappingHash string
}

// ReportCache holds the cached scan report with its content-hash token.
// Invariant: exactly one goroutine (the rescan loop) calls SetReport; multiple
// HandleConn goroutines read via snap.Load(). This single-writer/multi-reader
// pattern is safe with atomic.Pointer without CAS.
type ReportCache struct {
	snap atomic.Pointer[reportSnapshot]
}

// Token returns the published content-hash, or empty before the first scan.
func (c *ReportCache) Token() string {
	if c == nil {
		return ""
	}
	snap := c.snap.Load()
	if snap == nil {
		return ""
	}
	return snap.token
}

// SetReport atomically publishes a new report and facts. The token is an
// XXH64 of that content, so identical rescans stay unchanged for Sensor.
//
// mappingHash is the repo-to-CPE mapping the scan used, so GetReport can send
// it after a later Sync has already replaced the live mapping.
//
// r and facts are defensively copied so that a caller mutating its own copy
// after this call (or reusing the same facts map across scans) can never
// mutate the published, supposedly-immutable snapshot out from under
// concurrent readers.
func (c *ReportCache) SetReport(r *v4.IndexReport, facts map[string]string, mappingHash string) {
	cloned := cloneIndexReport(r)
	clonedFacts := cloneFacts(facts)
	c.snap.Store(&reportSnapshot{
		report:      cloned,
		token:       reportToken(cloned, clonedFacts),
		generatedAt: time.Now(),
		facts:       clonedFacts,
		mappingHash: mappingHash,
	})
}

// reportToken is XXH64 of the IndexReport and facts, as 16 lowercase hex
// digits. Deterministic marshal keeps identical content stable across rescans.
func reportToken(r *v4.IndexReport, facts map[string]string) string {
	h := xxhash.New()
	if r != nil {
		b, err := proto.MarshalOptions{Deterministic: true}.Marshal(r)
		if err == nil {
			_, _ = h.Write(b)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(facts)) {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(facts[k]))
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%016x", h.Sum64())
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
	provider     MappingProvider
	updater      MappingUpdater
}

// NewHandler creates a protocol handler. updater is nil for URL-managed
// agents, so SyncRepoCPEMapping is rejected with MAPPING_NOT_SENSOR_MANAGED.
func NewHandler(cache *ReportCache, agentVersion string, provider MappingProvider, updater MappingUpdater) *Handler {
	return &Handler{cache: cache, agentVersion: agentVersion, provider: provider, updater: updater}
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
		return
	}
}

func (h *Handler) dispatch(req *pb.VMServiceRequest) *pb.VMServiceResponse {
	switch req.GetMethod().(type) {
	case *pb.VMServiceRequest_GetReport:
		return h.handleGetReport(req.GetGetReport())
	case *pb.VMServiceRequest_SyncRepoCpeMapping:
		return h.handleSyncRepoCPEMapping(req.GetSyncRepoCpeMapping())
	default:
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD, "unknown or unset method")
	}
}

func (h *Handler) handleGetReport(req *pb.GetReportRequest) *pb.VMServiceResponse {
	if !h.provider.Ready() {
		log.Info("GetReport: no repo-to-CPE mapping available yet")
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_MAPPING_REQUIRED, "repository-to-CPE mapping not yet available")
	}

	snap := h.cache.snap.Load()
	if snap == nil || snap.report == nil {
		log.Info("GetReport: not ready (initial scan in progress)")
		return h.errorResponseFromSnap(snap, pb.ErrorCode_ERROR_CODE_NOT_READY, "initial scan in progress, try again later")
	}

	// Empty last_known_token forces a full report (first request or Sensor
	// refresh). A matching token is identical content; any other value
	// means the published report or facts changed.
	if lastKnown := req.GetLastKnownToken(); lastKnown != "" && lastKnown == snap.token {
		log.Infof("GetReport: unchanged (token=%s)", snap.token)
		resp := h.newResponseFromSnap(snap)
		resp.Result = &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{Unchanged: true},
		}
		return resp
	}

	log.Infof("GetReport: serving report (token=%s, packages=%d)", snap.token, len(snap.report.GetContents().GetPackages()))
	resp := h.newResponseFromSnap(snap)
	resp.Result = &pb.VMServiceResponse_GetReport{
		GetReport: &pb.GetReportResponse{IndexReport: snap.report},
	}
	return resp
}

func (h *Handler) newResponseFromSnap(snap *reportSnapshot) *pb.VMServiceResponse {
	var facts map[string]string
	var token string
	mappingHash := h.provider.Hash()
	if snap != nil {
		facts = cloneFacts(snap.facts)
		token = snap.token
		mappingHash = snap.mappingHash
	}
	updatePath := h.provider.UpdatePath()
	methods := []string{methodGetReport}
	if h.updater != nil {
		methods = append(methods, methodSyncRepoCPEMapping)
	}
	meta := &pb.ResponseMeta{
		AgentVersion:             h.agentVersion,
		ReportToken:              token,
		SupportedMethods:         methods,
		Facts:                    facts,
		RepoCpeMappingHash:       new(mappingHash),
		RepoCpeMappingUpdatePath: updatePath.Enum(),
	}
	if snap != nil && !snap.generatedAt.IsZero() {
		meta.ReportGeneratedAt = timestamppb.New(snap.generatedAt)
	}
	return &pb.VMServiceResponse{Meta: meta}
}

// handleSyncRepoCPEMapping applies a Sensor-pushed mapping. URL-managed
// agents (updater == nil) reject Sync outright; Update keeps the last-good
// mapping on a validation failure.
func (h *Handler) handleSyncRepoCPEMapping(req *pb.SyncRepoCPEMappingRequest) *pb.VMServiceResponse {
	if h.updater == nil {
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_MAPPING_NOT_SENSOR_MANAGED, "agent is URL-managed and does not accept pushed mappings")
	}
	updated, err := h.updater.Update(req.GetMapping())
	if err != nil {
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_MALFORMED_REQUEST, fmt.Sprintf("invalid repository-to-CPE mapping: %v", err))
	}
	resp := h.newResponseFromSnap(h.cache.snap.Load())
	resp.Result = &pb.VMServiceResponse_SyncRepoCpeMapping{
		SyncRepoCpeMapping: &pb.SyncRepoCPEMappingResponse{Updated: updated},
	}
	return resp
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
