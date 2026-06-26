package vsockclient

import (
	"errors"
	"fmt"
	"io"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/pkg/vsockframing"
	"google.golang.org/protobuf/proto"
)

// CapabilityReportV1 is the protocol capability for the v1 report exchange.
const CapabilityReportV1 = "report_v1"

var (
	// ErrNotReady indicates the agent has not yet generated a report.
	ErrNotReady = errors.New("agent has not yet generated a report")
	// ErrUnknownMethod indicates the agent does not support the requested method.
	ErrUnknownMethod = errors.New("agent does not support the requested method")
)

// GetReportResult holds the parsed response from a GetReport call.
type GetReportResult struct {
	IndexReport *v4.IndexReport
	Unchanged   bool
	Meta        *pb.ResponseMeta
}

// Client sends VMServiceRequests and reads VMServiceResponses over a framed stream.
type Client struct {
	capabilities    []string
	maxResponseSize int
}

// NewClient creates a protocol client with the given Sensor capabilities
// and maximum response size in bytes.
func NewClient(capabilities []string, maxResponseSize int) *Client {
	return &Client{capabilities: capabilities, maxResponseSize: maxResponseSize}
}

// GetReport sends a GetReportRequest and returns the response.
// The stream must be an io.ReadWriteCloser (from MultiDialer.Dial).
func (c *Client) GetReport(stream io.ReadWriteCloser, ifNewerThan uint32) (*GetReportResult, error) {
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{
			RequestId:    uuid.NewV4().String(),
			Capabilities: c.capabilities,
		},
		Method: &pb.VMServiceRequest_GetReport{
			GetReport: &pb.GetReportRequest{
				IfNewerThanGeneration: ifNewerThan,
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
	default:
		return fmt.Errorf("agent error (%s): %s", e.GetCode(), e.GetMessage())
	}
}
