package vsockserver

import (
	"errors"
	"net"
	"path/filepath"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
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

func getReportRequest(requestID, lastKnownToken string) *pb.VMServiceRequest {
	return &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: requestID},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{LastKnownToken: lastKnownToken}},
	}
}

func TestHandleRequest_GetReport(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil, "mapping-hash")
	provider := &fakeMappingProvider{ready: true, hash: "mapping-hash", updatePath: pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR}

	handler := NewHandler(cache, "test-1.0.0", provider, nil)
	req := getReportRequest("req-1", "")
	req.Meta.Capabilities = []string{"report_v1"}

	resp := sendAndReceive(t, handler, req)

	assert.NotNil(t, resp.GetGetReport())
	assert.Equal(t, "test-hash", resp.GetGetReport().GetIndexReport().GetHashId())
	assert.False(t, resp.GetGetReport().GetUnchanged())

	meta := resp.GetMeta()
	require.NotNil(t, meta)
	assert.Equal(t, "test-1.0.0", meta.GetAgentVersion())
	assert.NotEmpty(t, meta.GetReportToken())
	assert.NotNil(t, meta.GetReportGeneratedAt())
	assert.Contains(t, meta.GetSupportedMethods(), methodGetReport)
	assert.NotContains(t, meta.GetSupportedMethods(), methodSyncRepoCPEMapping)
	assert.Equal(t, "mapping-hash", meta.GetRepoCpeMappingHash())
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR, meta.GetRepoCpeMappingUpdatePath())
}

func TestHandleRequest_GetReport_MappingHashStaysWithTheReport(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil, "hash-at-scan")
	provider := &fakeMappingProvider{ready: true, hash: "hash-after-later-sync"}

	handler := NewHandler(cache, "test-1.0.0", provider, nil)
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-snap-hash"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	resp := sendAndReceive(t, handler, req)

	assert.Equal(t, "hash-at-scan", resp.GetMeta().GetRepoCpeMappingHash(),
		"GetReport must send the hash the cached report was built with, not the mapping that landed after the scan")
}

func TestHandleRequest_GetReport_Unchanged(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil, "")

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
	first := sendAndReceive(t, handler, getReportRequest("req-learn-token", ""))
	token := first.GetMeta().GetReportToken()
	require.NotEmpty(t, token)

	resp := sendAndReceive(t, handler, getReportRequest("req-2", token))

	assert.NotNil(t, resp.GetGetReport())
	assert.True(t, resp.GetGetReport().GetUnchanged())
	assert.Nil(t, resp.GetGetReport().GetIndexReport())
	assert.Equal(t, token, resp.GetMeta().GetReportToken())
}

// TestHandleRequest_GetReport_TokenMismatch covers a Sensor-cached token
// that does not match the agent's current content.
func TestHandleRequest_GetReport_TokenMismatch(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "post-restart-hash"}, nil, "")

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
	resp := sendAndReceive(t, handler, getReportRequest("req-mismatch", "stale-token"))

	assert.NotNil(t, resp.GetGetReport())
	assert.False(t, resp.GetGetReport().GetUnchanged())
	require.NotNil(t, resp.GetGetReport().GetIndexReport())
	assert.Equal(t, "post-restart-hash", resp.GetGetReport().GetIndexReport().GetHashId())
	assert.NotEmpty(t, resp.GetMeta().GetReportToken())
	assert.NotEqual(t, "stale-token", resp.GetMeta().GetReportToken())
}

// TestHandleRequest_GetReport_IdenticalRescanUnchanged covers a rescan
// whose report and facts did not change: the token is stable, so Sensor
// gets unchanged unless it sends an empty last_known_token.
func TestHandleRequest_GetReport_IdenticalRescanUnchanged(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "same-hash"}, nil, "")

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
	first := sendAndReceive(t, handler, getReportRequest("req-first-scan", ""))
	token := first.GetMeta().GetReportToken()
	require.NotEmpty(t, token)

	cache.SetReport(&v4.IndexReport{HashId: "same-hash"}, nil, "")
	resp := sendAndReceive(t, handler, getReportRequest("req-after-rescan", token))

	assert.NotNil(t, resp.GetGetReport())
	assert.True(t, resp.GetGetReport().GetUnchanged())
	assert.Nil(t, resp.GetGetReport().GetIndexReport())
	assert.Equal(t, token, resp.GetMeta().GetReportToken())
}

// TestHandleRequest_GetReport_ContentChangeServesReport covers a rescan
// whose content changed: the token changes and the full report is served.
func TestHandleRequest_GetReport_ContentChangeServesReport(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "before"}, nil, "")

	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)
	first := sendAndReceive(t, handler, getReportRequest("req-before", ""))
	token := first.GetMeta().GetReportToken()
	require.NotEmpty(t, token)

	cache.SetReport(&v4.IndexReport{HashId: "after"}, nil, "")
	resp := sendAndReceive(t, handler, getReportRequest("req-after", token))

	assert.NotNil(t, resp.GetGetReport())
	assert.False(t, resp.GetGetReport().GetUnchanged())
	assert.Equal(t, "after", resp.GetGetReport().GetIndexReport().GetHashId())
	assert.NotEmpty(t, resp.GetMeta().GetReportToken())
	assert.NotEqual(t, token, resp.GetMeta().GetReportToken())
}

func TestReportToken(t *testing.T) {
	report := &v4.IndexReport{HashId: "a"}
	assert.Equal(t, reportToken(report, nil), reportToken(&v4.IndexReport{HashId: "a"}, map[string]string{}))
	assert.NotEqual(t, reportToken(report, nil), reportToken(&v4.IndexReport{HashId: "b"}, nil))
	assert.NotEqual(t, reportToken(report, nil), reportToken(report, map[string]string{"os": "rhel"}))
	assert.Equal(t, reportToken(nil, nil), reportToken(nil, map[string]string{}))
	assert.Len(t, reportToken(report, nil), 16)
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
	cache.SetReport(&v4.IndexReport{HashId: "x"}, nil, "")
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
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil, "")
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
	assert.NotContains(t, meta.GetSupportedMethods(), methodSyncRepoCPEMapping)
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
		"oversize or otherwise invalid content is rejected as malformed": {
			updater:  &fakeMappingUpdater{err: errors.New("mapping size exceeds cap")},
			wantCode: pb.ErrorCode_ERROR_CODE_MALFORMED_REQUEST,
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
			assert.Contains(t, resp.GetMeta().GetSupportedMethods(), methodSyncRepoCPEMapping)
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

// TestHandleConn_GetReport_DoesNotApplyPending covers a rescan in flight
// while Sensor still polls: GetReport must not apply a deferred Sync. The
// scan itself applies pending when it returns.
func TestHandleConn_GetReport_DoesNotApplyPending(t *testing.T) {
	cache := &ReportCache{}
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	u := NewSensorUpdater(cachePath, "", nil)
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	scanHash := u.Hash()
	cache.SetReport(&v4.IndexReport{HashId: "gen-1"}, nil, scanHash)

	handler := NewHandler(cache, "test-1.0.0", u, u)
	u.MarkScanBusy()

	getReportReq := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-during-scan"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}
	resp := sendAndReceive(t, handler, getReportReq)
	require.NotNil(t, resp.GetGetReport())

	updated, err := u.Update([]byte(otherValidMappingJSON))
	require.NoError(t, err)
	require.True(t, updated)
	assert.Equal(t, scanHash, u.Hash(), "GetReport must not apply a deferred Sync")

	cache.SetReport(&v4.IndexReport{HashId: "gen-2"}, nil, scanHash)
	resp = sendAndReceive(t, handler, getReportReq)
	require.NotNil(t, resp.GetGetReport())
	assert.Equal(t, scanHash, u.Hash(), "GetReport of a newer cached report must still not apply pending")
	assert.Equal(t, scanHash, resp.GetMeta().GetRepoCpeMappingHash())

	u.MarkScanIdleAndApplyPending()
	assert.Equal(t, cpemapping.HashMapping([]byte(otherValidMappingJSON)), u.Hash())
	waitForCacheContent(t, u, cachePath, otherValidMappingJSON)

	resp = sendAndReceive(t, handler, getReportReq)
	assert.Equal(t, scanHash, resp.GetMeta().GetRepoCpeMappingHash(),
		"the cached report must still send the hash it was built with")
}

// TestHandleConn_UngatedProviderNeverPanics covers a URL-managed agent's
// provider, which does not implement ScanBusyGate.
func TestHandleConn_UngatedProviderNeverPanics(t *testing.T) {
	cache := &ReportCache{}
	cache.SetReport(&v4.IndexReport{HashId: "test-hash"}, nil, "")
	handler := NewHandler(cache, "test-1.0.0", readyProvider(), nil)

	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-ungated"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	}

	resp := sendAndReceive(t, handler, req)
	assert.NotNil(t, resp.GetGetReport())
}
