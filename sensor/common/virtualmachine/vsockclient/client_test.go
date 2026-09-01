package vsockclient

import (
	"context"
	"net"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// serveOnce plays the agent side of a request/response exchange over
// agentConn. Run in its own goroutine, since GetReport blocks on the
// response for the duration of the net.Pipe round trip.
func serveOnce(t *testing.T, agentConn net.Conn, resp *pb.VMServiceResponse, validateReq func(*pb.VMServiceRequest)) {
	defer utils.IgnoreError(agentConn.Close)
	reqData, err := vsockframing.ReadFrame(agentConn, 10<<20)
	require.NoError(t, err)

	if validateReq != nil {
		var req pb.VMServiceRequest
		require.NoError(t, proto.Unmarshal(reqData, &req))
		validateReq(&req)
	}

	respData, err := proto.Marshal(resp)
	require.NoError(t, err)
	require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
}

func TestSendGetReport_Success(t *testing.T) {
	client := NewClient([]string{CapabilityReportV1}, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer utils.IgnoreError(clientConn.Close)

	go serveOnce(t, agentConn, &pb.VMServiceResponse{
		Meta: &pb.ResponseMeta{AgentVersion: "test-agent", ReportToken: "tok-1"},
		Result: &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{
				IndexReport: &v4.IndexReport{HashId: "test-hash"},
			},
		},
	}, func(req *pb.VMServiceRequest) {
		assert.NotEmpty(t, req.GetMeta().GetRequestId())
		assert.Equal(t, []string{CapabilityReportV1}, req.GetMeta().GetCapabilities())
		assert.Empty(t, req.GetGetReport().GetLastKnownToken())
	})

	result, err := client.GetReport(context.Background(), clientConn, "")
	require.NoError(t, err)
	assert.Equal(t, "test-hash", result.IndexReport.GetHashId())
	assert.False(t, result.Unchanged)
	assert.Equal(t, "tok-1", result.Meta.GetReportToken())
}

func TestSendGetReport_Unchanged(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer utils.IgnoreError(clientConn.Close)

	go serveOnce(t, agentConn, &pb.VMServiceResponse{
		Meta: &pb.ResponseMeta{AgentVersion: "test-agent", ReportToken: "tok-5"},
		Result: &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{Unchanged: true},
		},
	}, func(req *pb.VMServiceRequest) {
		assert.Equal(t, "tok-5", req.GetGetReport().GetLastKnownToken())
	})

	result, err := client.GetReport(context.Background(), clientConn, "tok-5")
	require.NoError(t, err)
	assert.Nil(t, result.IndexReport)
	assert.True(t, result.Unchanged)
	assert.Equal(t, "tok-5", result.Meta.GetReportToken())
}

func TestSendGetReport_NilReportRejected(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer utils.IgnoreError(clientConn.Close)

	go serveOnce(t, agentConn, &pb.VMServiceResponse{
		Meta: &pb.ResponseMeta{AgentVersion: "test-agent", ReportToken: "tok-1"},
		Result: &pb.VMServiceResponse_GetReport{
			GetReport: &pb.GetReportResponse{},
		},
	}, nil)

	_, err := client.GetReport(context.Background(), clientConn, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IndexReport is nil")
}

func TestSendGetReport_ErrorCodes(t *testing.T) {
	cases := map[string]struct {
		code      pb.ErrorCode
		message   string
		wantErr   error
		wantInMsg string
	}{
		"should wrap ErrNotReady for NOT_READY": {
			code:    pb.ErrorCode_ERROR_CODE_NOT_READY,
			message: "report not yet generated",
			wantErr: ErrNotReady,
		},
		"should wrap ErrUnknownMethod for UNKNOWN_METHOD": {
			code:    pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD,
			message: "get_report not supported",
			wantErr: ErrUnknownMethod,
		},
		"should wrap ErrInternal for INTERNAL": {
			code:      pb.ErrorCode_ERROR_CODE_INTERNAL,
			message:   "scan crashed",
			wantErr:   ErrInternal,
			wantInMsg: "scan crashed",
		},
		"should wrap ErrMalformedRequest for MALFORMED_REQUEST": {
			code:      pb.ErrorCode_ERROR_CODE_MALFORMED_REQUEST,
			message:   "empty request_id",
			wantErr:   ErrMalformedRequest,
			wantInMsg: "empty request_id",
		},
		"should wrap ErrRequestTooLarge for REQUEST_TOO_LARGE": {
			code:      pb.ErrorCode_ERROR_CODE_REQUEST_TOO_LARGE,
			message:   "payload exceeds 10MB limit",
			wantErr:   ErrRequestTooLarge,
			wantInMsg: "10MB",
		},
		"should wrap ErrBusy for BUSY": {
			code:      pb.ErrorCode_ERROR_CODE_BUSY,
			message:   "agent is already serving another request",
			wantErr:   ErrBusy,
			wantInMsg: "another request",
		},
		"should wrap ErrUnknownAgentError for UNSPECIFIED": {
			code:    pb.ErrorCode_ERROR_CODE_UNSPECIFIED,
			message: "",
			wantErr: ErrUnknownAgentError,
		},
		"should wrap ErrMappingRequired for MAPPING_REQUIRED": {
			code:    pb.ErrorCode_ERROR_CODE_MAPPING_REQUIRED,
			message: "repository-to-CPE mapping not yet available",
			wantErr: ErrMappingRequired,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := NewClient(nil, 10<<20)
			clientConn, agentConn := net.Pipe()
			defer utils.IgnoreError(clientConn.Close)

			go serveOnce(t, agentConn, &pb.VMServiceResponse{
				Meta: &pb.ResponseMeta{AgentVersion: "test-agent"},
				Result: &pb.VMServiceResponse_Error{
					Error: &pb.ErrorResponse{Code: tc.code, Message: tc.message},
				},
			}, nil)

			_, err := client.GetReport(context.Background(), clientConn, "")
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
			if tc.wantInMsg != "" {
				assert.Contains(t, err.Error(), tc.wantInMsg)
			}
		})
	}
}

// Cancelling ctx must unblock a GetReport that is stuck waiting for a
// response — shutdown cancel does not rewrite the dial-time deadline, so
// GetReport closes the stream itself.
func TestSendGetReport_ContextCancelUnblocks(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer utils.IgnoreError(agentConn.Close)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Consume the request so GetReport is blocked in ReadFrame, then cancel.
	reqRead := make(chan struct{})
	go func() {
		defer close(reqRead)
		_, err := vsockframing.ReadFrame(agentConn, 10<<20)
		require.NoError(t, err)
	}()

	errCh := make(chan error, 1)
	go func() {
		_, err := client.GetReport(ctx, clientConn, "")
		errCh <- err
	}()

	select {
	case <-reqRead:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for GetReport to send request")
	}
	cancel()

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("GetReport did not unblock on context cancel")
	}
}
