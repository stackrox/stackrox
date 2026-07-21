package vsockclient

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
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
	// facts is populated into every request's RequestMeta.facts. Empty for
	// real Sensor traffic; only set by NewClient when hammer-mode load
	// testing is enabled (see below).
	facts map[string]string
}

// NewClient creates a protocol client with the given Sensor capabilities
// and maximum response size in bytes. maxResponseSize must fit in
// [1, math.MaxUint32], since it's narrowed to uint32 when passed to
// vsockframing.ReadFrame; out-of-range values (e.g. from a misconfigured
// size setting) fall back to defaultMaxResponseSize.
//
// LOAD-TEST ONLY: if ROX_VM_VSOCK_LOADTEST_HAMMER_MODE=true (never in a
// release build), every request's facts is also populated with report_size
// (from ROX_VM_VSOCK_LOADTEST_REPORT_SIZE, default "medium") so a fake
// roxagent can pick a canned report by size -- a no-op for real Sensor
// traffic.
func NewClient(capabilities []string, maxResponseSize int) *Client {
	if maxResponseSize <= 0 || int64(maxResponseSize) > math.MaxUint32 {
		maxResponseSize = defaultMaxResponseSize
	}
	c := &Client{capabilities: capabilities, maxResponseSize: maxResponseSize}
	if !buildinfo.ReleaseBuild && os.Getenv("ROX_VM_VSOCK_LOADTEST_HAMMER_MODE") == "true" {
		reportSize := cmp.Or(os.Getenv("ROX_VM_VSOCK_LOADTEST_REPORT_SIZE"), "medium")
		log.Warnf("LOAD TEST ONLY: ROX_VM_VSOCK_LOADTEST_HAMMER_MODE=true — requesting report_size=%q on every request", reportSize)
		c.facts = map[string]string{"report_size": reportSize}
	}
	return c
}

// GetReport sends a GetReportRequest and returns the response. knownEpoch is
// the last epoch Sensor observed for this VM (see ResponseMeta.epoch); pass 0
// when unknown. Sending it lets the agent detect a restart-coincidence false
// match and serve the full report in this same round trip, instead of the
// caller needing a second, forced request.
// The stream must be an io.ReadWriteCloser (from MultiDialer.Dial).
func (c *Client) GetReport(stream io.ReadWriteCloser, ifNewerThan uint32, knownEpoch uint32) (*GetReportResult, error) {
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{
			RequestId:    uuid.NewV4().String(),
			Capabilities: c.capabilities,
			Facts:        c.facts,
		},
		Method: &pb.VMServiceRequest_GetReport{
			GetReport: &pb.GetReportRequest{
				LastKnownGeneration: ifNewerThan,
				KnownEpoch:          knownEpoch,
			},
		},
	}

	reqData, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	if err := vsockframing.WriteFrame(stream, reqData); err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}

	respData, err := vsockframing.ReadFrame(stream, uint32(c.maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
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

func errorFromResponse(e *pb.ErrorResponse) error {
	switch e.GetCode() {
	case pb.ErrorCode_ERROR_CODE_NOT_READY:
		return fmt.Errorf("%w: %s", ErrNotReady, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD:
		return fmt.Errorf("%w: %s", ErrUnknownMethod, e.GetMessage())
	case pb.ErrorCode_ERROR_CODE_INTERNAL:
		return fmt.Errorf("%w: %s", ErrInternal, e.GetMessage())
	default:
		return fmt.Errorf("agent error (%s): %s", e.GetCode(), e.GetMessage())
	}
}
