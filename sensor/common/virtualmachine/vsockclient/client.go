package vsockclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/vsockframing"
	"google.golang.org/protobuf/proto"
)

var log = logging.LoggerForModule()

// CapabilityReportV1 is the protocol capability for the v1 report exchange.
const CapabilityReportV1 = "report_v1"

var (
	// ErrNotReady indicates the agent has not yet generated a report.
	ErrNotReady = errors.New("agent has not yet generated a report")
	// ErrUnknownMethod indicates the agent does not support the requested method.
	ErrUnknownMethod = errors.New("agent does not support the requested method")
	// ErrInternal indicates the agent encountered an internal error.
	ErrInternal = errors.New("agent internal error")
	// ErrMalformedRequest indicates the agent rejected the request as invalid.
	ErrMalformedRequest = errors.New("agent rejected request as malformed")
	// ErrRequestTooLarge indicates the request exceeded the agent's size limit.
	ErrRequestTooLarge = errors.New("agent rejected request as too large")
	// ErrBusy indicates the agent's single connection slot is held by another request.
	ErrBusy = errors.New("agent is busy with another request")
	// ErrUnknownAgentError indicates a well-formed ErrorResponse whose code
	// this client doesn't recognize (e.g. a future ErrorCode value).
	ErrUnknownAgentError = errors.New("unrecognized agent error code")
	// ErrMappingRequired indicates the agent has no repository-to-CPE mapping yet.
	ErrMappingRequired = errors.New("agent has no repository-to-CPE mapping yet")
)

// GetReportResult holds the parsed response from a GetReport call.
type GetReportResult struct {
	IndexReport *v4.IndexReport
	Unchanged   bool
	Meta        *pb.ResponseMeta
}

// defaultMaxResponseSize is the fallback used by NewClient for an
// out-of-range maxResponseSize. It matches the documented default of
// env.VirtualMachinesPullMaxResponseSizeKB (16 MiB): a misconfigured limit
// should fail toward a safe, bounded default, not toward "unlimited" —
// the data source is a VM agent, not a fully trusted peer.
const defaultMaxResponseSize = 16 << 20

// Client sends VMServiceRequests and reads VMServiceResponses over a framed stream.
type Client struct {
	capabilities    []string
	maxResponseSize int
}

// NewClient creates a protocol client with the given Sensor capabilities
// and maximum response size in bytes. maxResponseSize must fit in
// [1, math.MaxUint32], since it's narrowed to uint32 when passed to
// vsockframing.ReadFrame; out-of-range values (e.g. from a misconfigured
// size setting) fall back to defaultMaxResponseSize.
func NewClient(capabilities []string, maxResponseSize int) *Client {
	if maxResponseSize <= 0 || int64(maxResponseSize) > math.MaxUint32 {
		log.Warnf("VMScraper: configured max response size %d is out of range (0, %d], falling back to default of %d bytes",
			maxResponseSize, uint32(math.MaxUint32), defaultMaxResponseSize)
		maxResponseSize = defaultMaxResponseSize
	}
	return &Client{capabilities: capabilities, maxResponseSize: maxResponseSize}
}

// GetReport sends a GetReportRequest and returns the response. lastKnownToken
// is the last report_token Sensor observed for this VM; pass "" when unknown
// or when forcing a full report (mandatory refresh).
//
// The stream must be an io.ReadWriteCloser (from MultiDialer.Dial). If ctx is
// cancelled while a write or read is in progress, the stream is closed so the
// blocked I/O unblocks promptly — needed on Sensor shutdown, where parent
// cancel does not rewrite the dial-time socket deadline.
func (c *Client) GetReport(ctx context.Context, stream io.ReadWriteCloser, lastKnownToken string) (*GetReportResult, error) {
	stop := context.AfterFunc(ctx, func() {
		_ = stream.Close()
	})
	defer stop()

	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{
			RequestId:    uuid.NewV4().String(),
			Capabilities: c.capabilities,
		},
		Method: &pb.VMServiceRequest_GetReport{
			GetReport: &pb.GetReportRequest{
				LastKnownToken: lastKnownToken,
			},
		},
	}

	reqData, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	if err := vsockframing.WriteFrame(stream, reqData); err != nil {
		return nil, wrapStreamErr(ctx, "sending request", err)
	}

	respData, err := vsockframing.ReadFrame(stream, uint32(c.maxResponseSize))
	if err != nil {
		return nil, wrapStreamErr(ctx, "reading response", err)
	}

	var resp pb.VMServiceResponse
	if err := proto.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	switch r := resp.GetResult().(type) {
	case *pb.VMServiceResponse_GetReport:
		if r.GetReport.GetIndexReport() == nil && !r.GetReport.GetUnchanged() {
			return nil, errors.New("agent returned a new report response but IndexReport is nil")
		}
		return &GetReportResult{
			IndexReport: r.GetReport.GetIndexReport(),
			Unchanged:   r.GetReport.GetUnchanged(),
			Meta:        resp.GetMeta(),
		}, nil
	case *pb.VMServiceResponse_Error:
		return nil, errorFromResponse(r.Error)
	default:
		return nil, fmt.Errorf("unexpected response type: %T", resp.GetResult())
	}
}

func wrapStreamErr(ctx context.Context, op string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("%s: %w", op, ctxErr)
	}
	return fmt.Errorf("%s: %w", op, err)
}

func errorFromResponse(e *pb.ErrorResponse) error {
	switch e.GetCode() {
	case pb.ErrorCode_ERROR_CODE_NOT_READY:
		return fmt.Errorf("%w: %s", ErrNotReady, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD:
		return fmt.Errorf("%w: %s", ErrUnknownMethod, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_INTERNAL:
		return fmt.Errorf("%w: %s", ErrInternal, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_MALFORMED_REQUEST:
		return fmt.Errorf("%w: %s", ErrMalformedRequest, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_REQUEST_TOO_LARGE:
		return fmt.Errorf("%w: %s", ErrRequestTooLarge, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_BUSY:
		return fmt.Errorf("%w: %s", ErrBusy, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_MAPPING_REQUIRED:
		return fmt.Errorf("%w: %s", ErrMappingRequired, e.GetMessage())
	default:
		return fmt.Errorf("%w: agent error (%s): %s", ErrUnknownAgentError, e.GetCode(), e.GetMessage())
	}
}
