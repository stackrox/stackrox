package vsockclient

import (
	"net"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestSendGetReport_Success(t *testing.T) {
	client := NewClient([]string{CapabilityReportV1}, 10<<20)
	clientConn, agentConn := net.Pipe()
	defer utils.IgnoreError(clientConn.Close)

	go func() {
		defer utils.IgnoreError(agentConn.Close)
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
	defer utils.IgnoreError(clientConn.Close)

	go func() {
		defer utils.IgnoreError(agentConn.Close)
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
	defer utils.IgnoreError(clientConn.Close)

	go func() {
		defer utils.IgnoreError(agentConn.Close)
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
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			client := NewClient(nil, 10<<20)
			clientConn, agentConn := net.Pipe()
			defer utils.IgnoreError(clientConn.Close)

			go func() {
				defer utils.IgnoreError(agentConn.Close)
				_, err := vsockframing.ReadFrame(agentConn, 10<<20)
				require.NoError(t, err)

				resp := &pb.VMServiceResponse{
					Meta: &pb.ResponseMeta{AgentVersion: "test-agent"},
					Result: &pb.VMServiceResponse_Error{
						Error: &pb.ErrorResponse{Code: tc.code, Message: tc.message},
					},
				}
				respData, err := proto.Marshal(resp)
				require.NoError(t, err)
				require.NoError(t, vsockframing.WriteFrame(agentConn, respData))
			}()

			_, err := client.GetReport(clientConn, 0)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.wantErr)
			if tc.wantInMsg != "" {
				assert.Contains(t, err.Error(), tc.wantInMsg)
			}
		})
	}
}
