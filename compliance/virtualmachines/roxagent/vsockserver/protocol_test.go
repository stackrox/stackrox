package vsockserver

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

func sendAndReceive(t *testing.T, handler *Handler, req *pb.VMServiceRequest) *pb.VMServiceResponse {
	t.Helper()
	clientConn, serverConn := net.Pipe()
	go handler.HandleConn(serverConn)

	reqData, err := proto.Marshal(req)
	require.NoError(t, err)
	require.NoError(t, vsockframing.WriteFrame(clientConn, reqData))

	respData, err := vsockframing.ReadFrame(clientConn, 10<<20)
	require.NoError(t, err)
	_ = clientConn.Close()

	var resp pb.VMServiceResponse
	require.NoError(t, proto.Unmarshal(respData, &resp))
	return &resp
}

func TestHandleRequest_GetReport(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-1", Capabilities: []string{"report_v1"}},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{IfNewerThanGeneration: 0}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.Equal(t, "test-hash", resp.GetGetReport().GetIndexReport().GetHashId())
	assert.False(t, resp.GetGetReport().GetUnchanged())

	meta := resp.GetMeta()
	require.NotNil(t, meta)
	assert.Equal(t, "test-1.0.0", meta.GetAgentVersion())
	assert.Equal(t, uint32(1), meta.GetReportGeneration())
	assert.NotNil(t, meta.GetReportGeneratedAt())
	assert.Contains(t, meta.GetSupportedMethods(), "get_report")
}

func TestHandleRequest_GetReport_Unchanged(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-2"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{IfNewerThanGeneration: 1}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.True(t, resp.GetGetReport().GetUnchanged())
	assert.Nil(t, resp.GetGetReport().GetIndexReport())
}

func TestHandleRequest_GetReport_GenerationRegression(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "post-restart-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-regression"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{
			IfNewerThanGeneration: 5,
		}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.False(t, resp.GetGetReport().GetUnchanged(), "agent restarted (gen=1 < requested=5), must serve full report")
	assert.Equal(t, "post-restart-hash", resp.GetGetReport().GetIndexReport().GetHashId())
	assert.Equal(t, uint32(1), resp.GetMeta().GetReportGeneration())
}

func TestHandleRequest_NotReady(t *testing.T) {
	cache := &ReportCache{}
	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-3"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.Nil(t, resp.GetGetReport())
	require.NotNil(t, resp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_NOT_READY, resp.GetError().GetCode())
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "x"}, nil)
	handler := NewHandler(cache, "test-1.0.0")

	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-4"},
		// Method oneof not set.
	}

	resp := sendAndReceive(t, handler, req)

	assert.Nil(t, resp.GetGetReport())
	require.NotNil(t, resp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD, resp.GetError().GetCode())
}
