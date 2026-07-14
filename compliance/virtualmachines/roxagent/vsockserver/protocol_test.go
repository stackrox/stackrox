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
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{LastKnownGeneration: 0}},
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
	assert.NotZero(t, meta.GetEpoch(), "epoch should be seeded on handler creation")
}

func TestHandleRequest_GetReport_Unchanged(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-2"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{LastKnownGeneration: 1}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.True(t, resp.GetGetReport().GetUnchanged())
	assert.Nil(t, resp.GetGetReport().GetIndexReport())
}

func TestHandleRequest_GetReport_UnchangedWhenKnownEpochMatches(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	// Learn the handler's epoch from a first exchange (known_epoch=0, so
	// Sensor has no cached epoch yet — falls back to generation-only).
	firstResp := sendAndReceive(t, handler, &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-learn-epoch"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{LastKnownGeneration: 0}},
	})
	epoch := firstResp.GetMeta().GetEpoch()
	require.NotZero(t, epoch)

	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-epoch-match"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{
			LastKnownGeneration: 1,
			KnownEpoch:          epoch,
		}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.True(t, resp.GetGetReport().GetUnchanged(), "matching generation and epoch should report unchanged")
	assert.Nil(t, resp.GetGetReport().GetIndexReport())
}

// TestHandleRequest_GetReport_ServesFullReportOnKnownEpochMismatch covers
// the case report_generation alone cannot distinguish: report_generation
// resets to 1 on every roxagent restart, so a restarted agent can
// coincidentally match a generation Sensor already has cached for a
// previous instance. known_epoch lets the agent detect this itself, in a
// single round trip, by comparing Sensor's last-seen epoch against its own
// current one.
func TestHandleRequest_GetReport_ServesFullReportOnKnownEpochMismatch(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "post-restart-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-epoch-mismatch"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{
			// Generation matches (both are 1), but Sensor's cached epoch is
			// from a previous agent process instance.
			LastKnownGeneration: 1,
			KnownEpoch:          12345,
		}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.False(t, resp.GetGetReport().GetUnchanged(), "epoch mismatch must serve the full report despite matching generation")
	require.NotNil(t, resp.GetGetReport().GetIndexReport())
	assert.Equal(t, "post-restart-hash", resp.GetGetReport().GetIndexReport().GetHashId())
	assert.NotEqual(t, uint32(12345), resp.GetMeta().GetEpoch(), "response should carry the agent's real current epoch")
}

// TestHandleRequest_GetReport_UnchangedWhenKnownEpochZero pins down backward
// compatibility: known_epoch=0 means Sensor has no epoch to compare (first
// request for this VM, or a Sensor build that predates the field), so the
// agent must fall back to generation-only comparison exactly as before this
// field existed.
func TestHandleRequest_GetReport_UnchangedWhenKnownEpochZero(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-epoch-zero"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{
			LastKnownGeneration: 1,
			KnownEpoch:          0,
		}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.True(t, resp.GetGetReport().GetUnchanged(), "known_epoch=0 should fall back to generation-only comparison")
	assert.Nil(t, resp.GetGetReport().GetIndexReport())
}

func TestHandleRequest_GetReport_GenerationRegression(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "post-restart-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0")
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-regression"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{
			LastKnownGeneration: 5,
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
