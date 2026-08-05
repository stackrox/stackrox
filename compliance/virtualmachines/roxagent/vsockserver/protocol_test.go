package vsockserver

import (
	"errors"
	"net"
	"testing"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// fakeMappingProvider is a MappingProvider test double whose Ready/Hash/
// UpdatePath are directly settable; Bytes/Path are unused by the Handler
// and always error.
type fakeMappingProvider struct {
	ready      bool
	hash       string
	updatePath pb.RepoCPEMappingUpdatePath
}

func (f *fakeMappingProvider) Ready() bool                             { return f.ready }
func (f *fakeMappingProvider) Hash() string                            { return f.hash }
func (f *fakeMappingProvider) UpdatePath() pb.RepoCPEMappingUpdatePath { return f.updatePath }
func (f *fakeMappingProvider) Bytes() ([]byte, error)                  { return nil, errors.New("not implemented") }
func (f *fakeMappingProvider) Path() (string, error)                   { return "", errors.New("not implemented") }

var _ MappingProvider = (*fakeMappingProvider)(nil)

// fakeGatedProvider adds ScanBusyGate call tracking on top of
// fakeMappingProvider, mirroring SensorUpdater (gated) as opposed to
// URLUpdater (ungated) so tests can verify HandleConn only invokes the
// gate for providers that implement it.
type fakeGatedProvider struct {
	fakeMappingProvider
	busyCalls int
	idleCalls int
}

func (f *fakeGatedProvider) MarkScanBusy()                { f.busyCalls++ }
func (f *fakeGatedProvider) MarkScanIdleAndApplyPending() { f.idleCalls++ }

var (
	_ MappingProvider = (*fakeGatedProvider)(nil)
	_ ScanBusyGate    = (*fakeGatedProvider)(nil)
)

// fakeMappingUpdater is a MappingUpdater test double that returns a fixed
// (updated, err) pair and records the content it was called with.
type fakeMappingUpdater struct {
	updated bool
	err     error
	calls   [][]byte
}

func (f *fakeMappingUpdater) Update(content []byte) (bool, error) {
	f.calls = append(f.calls, content)
	return f.updated, f.err
}

var _ MappingUpdater = (*fakeMappingUpdater)(nil)

// readyProvider is the default fakeMappingProvider for GetReport test
// cases that aren't exercising mapping-readiness behavior itself.
func readyProvider() *fakeMappingProvider { return &fakeMappingProvider{ready: true} }

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
	provider := &fakeMappingProvider{ready: true, hash: "mapping-hash", updatePath: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR}

	handler := NewHandler(cache, "test-1.0.0", provider, nil)
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
	assert.Contains(t, meta.GetSupportedMethods(), "sync_repo_cpe_mapping")
	assert.NotZero(t, meta.GetEpoch(), "epoch should be seeded on handler creation")
	assert.Equal(t, "mapping-hash", meta.GetRepoCpeMappingHash())
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR, meta.GetRepoCpeMappingUpdatePath())
}

func TestHandleRequest_GetReport_Unchanged(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
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

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
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

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
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

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
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

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
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
	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
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
	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)

	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "req-4"},
		// Method oneof not set.
	}

	resp := sendAndReceive(t, handler, req)

	assert.Nil(t, resp.GetGetReport())
	require.NotNil(t, resp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD, resp.GetError().GetCode())
}

func TestHandleRequest_GetReport_MappingRequired(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)
	handler := NewHandler(cache, "test-1.0.0", &fakeMappingProvider{ready: false}, nil)

	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-mapping-required"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.Nil(t, resp.GetGetReport())
	require.NotNil(t, resp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_MAPPING_REQUIRED, resp.GetError().GetCode())
}

// TestHandleRequest_MetadataAlwaysPresentOnError pins down that an empty
// mapping hash is still an explicitly-present optional field, not a nil
// one Sensor would have to special-case.
func TestHandleRequest_MetadataAlwaysPresentOnError(t *testing.T) {
	cache := &ReportCache{}
	handler := NewHandler(cache, "test-1.0.0", &fakeMappingProvider{ready: false}, nil)

	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-meta-on-error"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	resp := sendAndReceive(t, handler, req)

	require.NotNil(t, resp.GetError())
	meta := resp.GetMeta()
	require.NotNil(t, meta)
	require.NotNil(t, meta.RepoCpeMappingHash, "hash must be present-but-empty, not nil")
	assert.Empty(t, meta.GetRepoCpeMappingHash())
	require.NotNil(t, meta.RepoCpeMappingUpdatePath)
	assert.Contains(t, meta.GetSupportedMethods(), "sync_repo_cpe_mapping")
}

func TestHandleRequest_SyncRepoCPEMapping(t *testing.T) {
	tests := map[string]struct {
		updater     *fakeMappingUpdater
		wantCode    pb.ErrorCode
		wantUpdated bool
	}{
		"new mapping content is applied": {
			updater:     &fakeMappingUpdater{updated: true},
			wantUpdated: true,
		},
		"unchanged mapping content is a no-op": {
			updater:     &fakeMappingUpdater{updated: false},
			wantUpdated: false,
		},
		"oversize or otherwise invalid content is rejected as internal error": {
			updater:  &fakeMappingUpdater{err: errors.New("mapping size exceeds cap")},
			wantCode: pb.ErrorCode_ERROR_CODE_INTERNAL,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cache := &ReportCache{}
			handler := NewHandler(cache, "test-1.0.0", readyProvider(), tt.updater)
			req := &pb.VMServiceRequest{
				Meta:   &pb.RequestMeta{RequestId: "req-sync"},
				Method: &pb.VMServiceRequest_SyncRepoCpeMapping{SyncRepoCpeMapping: &pb.SyncRepoCPEMappingRequest{Mapping: []byte("content")}},
			}

			resp := sendAndReceive(t, handler, req)

			if tt.wantCode != pb.ErrorCode_ERROR_CODE_UNSPECIFIED {
				require.NotNil(t, resp.GetError())
				assert.Equal(t, tt.wantCode, resp.GetError().GetCode())
				assert.Nil(t, resp.GetSyncRepoCpeMapping())
				return
			}
			require.NotNil(t, resp.GetSyncRepoCpeMapping())
			assert.Equal(t, tt.wantUpdated, resp.GetSyncRepoCpeMapping().GetUpdated())
			require.Len(t, tt.updater.calls, 1)
			assert.Equal(t, []byte("content"), tt.updater.calls[0])
		})
	}
}

func TestHandleRequest_SyncRepoCPEMapping_NilUpdater_NotSensorManaged(t *testing.T) {
	cache := &ReportCache{}
	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-sync-url-managed"},
		Method: &pb.VMServiceRequest_SyncRepoCpeMapping{SyncRepoCpeMapping: &pb.SyncRepoCPEMappingRequest{Mapping: []byte("content")}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.Nil(t, resp.GetSyncRepoCpeMapping())
	require.NotNil(t, resp.GetError())
	assert.Equal(t, pb.ErrorCode_ERROR_CODE_MAPPING_NOT_SENSOR_MANAGED, resp.GetError().GetCode())
}

// TestHandleConn_MarksProviderIdleAfterEveryResponse covers the deferred-
// apply path end-to-end: a rescanner marking the provider busy before a
// scan (simulated here directly) must see it cleared once - and only once
// - HandleConn has actually written the response for that round trip, so
// a Sync that arrived mid-scan can finally be promoted.
func TestHandleConn_MarksProviderIdleAfterEveryResponse(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)
	provider := &fakeGatedProvider{fakeMappingProvider: fakeMappingProvider{ready: true}}
	handler := NewHandler(cache, "test-1.0.0", provider, nil)

	provider.MarkScanBusy()
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-deferred-apply"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	sendAndReceive(t, handler, req)

	// HandleConn's post-write call happens after the client has already read
	// the response (see sendAndReceive), so it may not have run yet here.
	assert.Eventually(t, func() bool { return provider.idleCalls == 1 }, time.Second, time.Millisecond,
		"idle-and-apply-pending must fire exactly once, after the response is sent")
}

// TestHandleConn_UngatedProviderNeverPanics covers a URL-managed agent's
// provider, which does not implement ScanBusyGate: HandleConn's post-write
// type assertion must be a silent no-op instead of a panic.
func TestHandleConn_UngatedProviderNeverPanics(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil)
	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)

	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-ungated"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	resp := sendAndReceive(t, handler, req)
	assert.NotNil(t, resp.GetGetReport())
}
