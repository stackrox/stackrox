package vsockclient

import (
	"net"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSendGetReport_Success(t *testing.T) {
	client := NewClient([]string{CapabilityReportV1}, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	go func() {
		defer func() { _ = agentConn.Close() }()
		reqData, err := vsockframing.ReadFrame(agentConn, 10<<20)
		require.NoError(t, err)

		var req pb.VMServiceRequest
		require.NoError(t, proto.Unmarshal(reqData, &req))
		assert.NotEmpty(t, req.GetMeta().GetRequestId())
		assert.Equal(t, []string{CapabilityReportV1}, req.GetMeta().GetCapabilities())
		assert.Equal(t, uint32(0), req.GetGetReport().GetIfNewerThanGeneration())

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{AgentVersion: "test-agent", ReportGeneration: 1},
			Result: &pb.VMServiceResponse_GetReport{
				GetReport: &pb.GetReportResponse{
					IndexReport: &v4.IndexReport{HashId: "test-hash"},
				},
			},
		}
		respData, err := proto.Marshal(resp)
		require.NoError(t, err)
		require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
	}()

	result, err := client.GetReport(clientConn, 0)
	require.NoError(t, err)
	assert.Equal(t, "test-hash", result.IndexReport.GetHashId())
	assert.False(t, result.Unchanged)
	assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
}

func TestSendGetReport_Unchanged(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	go func() {
		defer func() { _ = agentConn.Close() }()
		_, err := vsockframing.ReadFrame(agentConn, 10<<20)
		require.NoError(t, err)

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{AgentVersion: "test-agent", ReportGeneration: 5},
			Result: &pb.VMServiceResponse_GetReport{
				GetReport: &pb.GetReportResponse{Unchanged: true},
			},
		}
		respData, err := proto.Marshal(resp)
		require.NoError(t, err)
		require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
	}()

	result, err := client.GetReport(clientConn, 5)
	require.NoError(t, err)
	assert.Nil(t, result.IndexReport)
	assert.True(t, result.Unchanged)
	assert.Equal(t, uint32(5), result.Meta.GetReportGeneration())
}

func TestSendGetReport_NilReportRejected(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	go func() {
		defer func() { _ = agentConn.Close() }()
		_, err := vsockframing.ReadFrame(agentConn, 10<<20)
		require.NoError(t, err)

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{AgentVersion: "test-agent", ReportGeneration: 1},
			Result: &pb.VMServiceResponse_GetReport{
				GetReport: &pb.GetReportResponse{},
			},
		}
		respData, err := proto.Marshal(resp)
		require.NoError(t, err)
		require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
	}()

	_, err := client.GetReport(clientConn, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IndexReport is nil")
}

func TestSendGetReport_NotReady(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	go func() {
		defer func() { _ = agentConn.Close() }()
		_, err := vsockframing.ReadFrame(agentConn, 10<<20)
		require.NoError(t, err)

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{AgentVersion: "test-agent"},
			Result: &pb.VMServiceResponse_Error{
				Error: &pb.ErrorResponse{
					Code:    pb.ErrorCode_ERROR_CODE_NOT_READY,
					Message: "report not yet generated",
				},
			},
		}
		respData, err := proto.Marshal(resp)
		require.NoError(t, err)
		require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
	}()

	_, err := client.GetReport(clientConn, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotReady)
}

func TestSendGetReport_UnknownMethod(t *testing.T) {
	client := NewClient(nil, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	go func() {
		defer func() { _ = agentConn.Close() }()
		_, err := vsockframing.ReadFrame(agentConn, 10<<20)
		require.NoError(t, err)

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{AgentVersion: "test-agent"},
			Result: &pb.VMServiceResponse_Error{
				Error: &pb.ErrorResponse{
					Code:    pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD,
					Message: "get_report not supported",
				},
			},
		}
		respData, err := proto.Marshal(resp)
		require.NoError(t, err)
		require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
	}()

	_, err := client.GetReport(clientConn, 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownMethod)
}
