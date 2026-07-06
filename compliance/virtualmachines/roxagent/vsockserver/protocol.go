package vsockserver

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxRequestSize = 1 << 20 // 1 MiB

// Handler processes incoming VSOCK protocol requests: frame reading/writing,
// method dispatch, and response construction. Where the report data in a
// response actually comes from is delegated to a ReportProvider (see
// report_provider.go) so this file stays purely about the wire protocol.
type Handler struct {
	provider     ReportProvider
	agentVersion string
}

// NewHandler creates a protocol handler backed by provider — either
// NewCacheReportProvider (production) or NewFakeReportProvider (load test).
func NewHandler(provider ReportProvider, agentVersion string) *Handler {
	return &Handler{provider: provider, agentVersion: agentVersion}
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
		return h.handleGetReport(req.GetGetReport(), req.GetMeta())
	default:
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD, "unknown or unset method")
	}
}

func (h *Handler) handleGetReport(req *pb.GetReportRequest, meta *pb.RequestMeta) *pb.VMServiceResponse {
	report, facts, generation, epoch, generatedAt, ready := h.provider.GetReport(req, meta)
	if !ready {
		log.Info("GetReport: not ready (initial scan in progress)")
		return h.errorResponse(pb.ErrorCode_ERROR_CODE_NOT_READY, "initial scan in progress, try again later")
	}

	resp := h.buildResponse(facts, generation, epoch, generatedAt)

	// Strict equality (not >=) so that after an agent restart — when the generation
	// counter resets to 1 — a sensor still holding a higher generation from the
	// previous instance will receive the full report instead of a false "unchanged".
	//
	// known_epoch guards against the opposite false positive: a restarted agent
	// whose reset generation coincidentally re-matches last_known_generation.
	// 0 means Sensor doesn't know our epoch yet (first-ever request for this VM,
	// or a Sensor build that predates the field), so fall back to generation-only
	// comparison exactly as before this field existed.
	generationMatches := req.GetLastKnownGeneration() == generation
	knownEpoch := req.GetKnownEpoch()
	epochMatches := knownEpoch == 0 || knownEpoch == epoch
	if generationMatches && epochMatches {
		log.Infof("GetReport: unchanged (generation=%d, last_known_generation=%d)", generation, req.GetLastKnownGeneration())
		resp.Result = &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{Unchanged: true},
		}
		return resp
	}
	if generationMatches && !epochMatches {
		log.Infof("GetReport: generation matches (%d) but epoch changed (known=%d, current=%d) — agent restarted, serving full report in this round trip",
			generation, knownEpoch, epoch)
	}

	log.Infof("GetReport: serving report (generation=%d, packages=%d)", generation, len(report.GetContents().GetPackages()))
	resp.Result = &pb.VMServiceResponse_GetReport{
		GetReport: &pb.GetReportResponse{IndexReport: report},
	}
	return resp
}

func (h *Handler) newResponse() *pb.VMServiceResponse {
	_, facts, generation, epoch, generatedAt, _ := h.provider.GetReport(nil, nil)
	return h.buildResponse(facts, generation, epoch, generatedAt)
}

func (h *Handler) buildResponse(facts map[string]string, generation uint32, epoch uint32, generatedAt time.Time) *pb.VMServiceResponse {
	if facts == nil {
		facts = map[string]string{}
	}
	meta := &pb.ResponseMeta{
		AgentVersion:     h.agentVersion,
		ReportGeneration: generation,
		SupportedMethods: []string{"get_report"},
		Facts:            facts,
		Epoch:            epoch,
	}
	if !generatedAt.IsZero() {
		meta.ReportGeneratedAt = timestamppb.New(generatedAt)
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
	_, ok := errors.AsType[tls.RecordHeaderError](err)
	return ok
}
