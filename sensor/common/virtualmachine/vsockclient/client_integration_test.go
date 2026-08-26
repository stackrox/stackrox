package vsockclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	roxagentvsock "github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
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

var _ roxagentvsock.MappingProvider = (*fakeMappingProvider)(nil)

// exchangeTimeout bounds a single client/handler exchange. Under synctest the
// fake clock advances this duration instantly once every goroutine is durably
// blocked (e.g. a protocol deadlock on net.Pipe), so GetReport's context
// cancel closes the stream via AfterFunc and the test fails fast instead of
// hanging the package.
const exchangeTimeout = 5 * time.Second

// protocolHarness drives a real Sensor client against a real roxagent handler
// over net.Pipe(), validating the protocol contract end-to-end.
type protocolHarness struct {
	client  *Client
	cache   *roxagentvsock.ReportCache
	handler *roxagentvsock.Handler
}

type protocolHarnessOptions struct {
	capabilities    []string
	maxResponseSize int
	agentVersion    string
	seedReport      *v4.IndexReport
	seedFacts       map[string]string
}

func newProtocolHarness(t *testing.T, opts protocolHarnessOptions) *protocolHarness {
	t.Helper()

	if opts.capabilities == nil {
		opts.capabilities = []string{CapabilityReportV1}
	}
	if opts.maxResponseSize == 0 {
		opts.maxResponseSize = 10 << 20
	}
	if opts.agentVersion == "" {
		opts.agentVersion = "test-agent"
	}

	cache := &roxagentvsock.ReportCache{}
	if opts.seedReport != nil {
		cache.SetReport(opts.seedReport, opts.seedFacts, "")
	}

	return &protocolHarness{
		client:  NewClient(opts.capabilities, opts.maxResponseSize),
		cache:   cache,
		handler: roxagentvsock.NewHandler(cache, opts.agentVersion, &fakeMappingProvider{ready: true}, nil),
	}
}

func (h *protocolHarness) getReport(t *testing.T, lastKnownToken string) (*GetReportResult, error) {
	t.Helper()
	return exchangeOnce(t, h.client, lastKnownToken, h.handler.HandleConn)
}

// exchangeOnce runs a single GetReport against responder over net.Pipe()
// inside a synctest bubble. A stuck peer makes the exchange timeout fire on
// the fake clock (via GetReport's AfterFunc stream close) instead of hanging
// the package on wall time.
func exchangeOnce(t *testing.T, client *Client, lastKnownToken string, responder func(net.Conn)) (*GetReportResult, error) {
	t.Helper()

	var result *GetReportResult
	var err error
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), exchangeTimeout)
		defer cancel()

		clientConn, agentConn := net.Pipe()

		done := make(chan struct{})
		go func() {
			defer close(done)
			responder(agentConn)
		}()

		result, err = client.GetReport(ctx, clientConn, lastKnownToken)
		// Close the client end before waiting so a responder blocked in WriteFrame
		// (or similar) is unblocked even when GetReport already returned.
		_ = clientConn.Close()
		select {
		case <-done:
		case <-ctx.Done():
			if err == nil {
				err = fmt.Errorf("waiting for responder: %w", ctx.Err())
			}
		}
	})
	return result, err
}

func TestGetReportIntegration(t *testing.T) {
	cases := map[string]struct {
		// Arguments
		seedReport     *v4.IndexReport
		seedFacts      map[string]string
		agentVersion   string
		lastKnownToken string
		// Expectations
		wantErr     error
		checkResult func(t *testing.T, result *GetReportResult)
	}{
		"should return not ready when cache is empty": {
			wantErr: ErrNotReady,
		},
		"should return full report when cache has report": {
			seedReport:   &v4.IndexReport{HashId: "integration-test-hash"},
			seedFacts:    map[string]string{"os": "rhel", "arch": "x86_64"},
			agentVersion: "roxagent-0.3.1-deadbeef",
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "integration-test-hash", result.IndexReport.GetHashId())

				require.NotNil(t, result.Meta)
				assert.Equal(t, "roxagent-0.3.1-deadbeef", result.Meta.GetAgentVersion())
				assert.NotEmpty(t, result.Meta.GetReportToken())
				assert.NotNil(t, result.Meta.GetReportGeneratedAt())
				assert.Contains(t, result.Meta.GetSupportedMethods(), "get_report")
				assert.Equal(t, "rhel", result.Meta.GetFacts()["os"])
				assert.Equal(t, "x86_64", result.Meta.GetFacts()["arch"])
			},
		},
		"should return full report when last known token does not match": {
			seedReport:     &v4.IndexReport{HashId: "post-restart-hash"},
			lastKnownToken: "stale-token",
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "post-restart-hash", result.IndexReport.GetHashId())
				assert.NotEmpty(t, result.Meta.GetReportToken())
				assert.NotEqual(t, "stale-token", result.Meta.GetReportToken())
				assert.NotNil(t, result.Meta.GetReportGeneratedAt())
				assert.Contains(t, result.Meta.GetSupportedMethods(), "get_report")
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			h := newProtocolHarness(t, protocolHarnessOptions{
				seedReport:   tc.seedReport,
				seedFacts:    tc.seedFacts,
				agentVersion: tc.agentVersion,
			})

			result, err := h.getReport(t, tc.lastKnownToken)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			if tc.checkResult != nil {
				tc.checkResult(t, result)
			}
		})
	}
}

// TestGetReportIntegration_TokenRoundTrip drives a real Client against a
// real roxagent Handler: matching token is unchanged, a different token
// (restart or new scan) serves the full report in the same round trip.
func TestGetReportIntegration_TokenRoundTrip(t *testing.T) {
	h := newProtocolHarness(t, protocolHarnessOptions{
		seedReport: &v4.IndexReport{HashId: "token-test-hash"},
	})

	first, err := h.getReport(t, "")
	require.NoError(t, err)
	require.False(t, first.Unchanged)
	token := first.Meta.GetReportToken()
	require.NotEmpty(t, token)

	t.Run("unchanged in a single round trip when last known token matches", func(t *testing.T) {
		result, err := h.getReport(t, token)
		require.NoError(t, err)
		assert.True(t, result.Unchanged)
		assert.Nil(t, result.IndexReport)
		assert.Equal(t, token, result.Meta.GetReportToken())
	})

	t.Run("full report in a single round trip when last known token mismatches", func(t *testing.T) {
		result, err := h.getReport(t, "other-token")
		require.NoError(t, err)
		assert.False(t, result.Unchanged)
		require.NotNil(t, result.IndexReport)
		assert.Equal(t, "token-test-hash", result.IndexReport.GetHashId())
		assert.Equal(t, token, result.Meta.GetReportToken())
	})
}

// syncOnce mirrors exchangeOnce for a single SyncRepoCPEMapping exchange.
func syncOnce(t *testing.T, client *Client, mapping []byte, responder func(net.Conn)) (bool, *pb.ResponseMeta, error) {
	t.Helper()

	var updated bool
	var meta *pb.ResponseMeta
	var err error
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), exchangeTimeout)
		defer cancel()

		clientConn, agentConn := net.Pipe()

		done := make(chan struct{})
		go func() {
			defer close(done)
			responder(agentConn)
		}()

		updated, meta, err = client.SyncRepoCPEMapping(ctx, clientConn, mapping)
		_ = clientConn.Close()
		select {
		case <-done:
		case <-ctx.Done():
			if err == nil {
				err = fmt.Errorf("waiting for responder: %w", ctx.Err())
			}
		}
	})
	return updated, meta, err
}

// TestSyncRepoCPEMappingIntegration_FullRoundTrip drives a real Client and
// Handler through GetReport (MAPPING_REQUIRED) -> SyncRepoCPEMapping ->
// GetReport again; every seam here is otherwise only unit-tested against fakes.
func TestSyncRepoCPEMappingIntegration_FullRoundTrip(t *testing.T) {
	mapping := []byte(`{"data":{"rhel-9-server-rpms":{"cpes":["cpe:/o:redhat:enterprise_linux:9"]}}}`)
	updater := roxagentvsock.NewSensorUpdater(filepath.Join(t.TempDir(), "cache.json"), "", nil)
	cache := &roxagentvsock.ReportCache{}
	handler := roxagentvsock.NewHandler(cache, "test-agent", updater, updater)
	client := NewClient([]string{CapabilityReportV1}, 10<<20)

	first, err := exchangeOnce(t, client, "", handler.HandleConn)
	require.ErrorIs(t, err, ErrMappingRequired, "no bootstrapped mapping must surface as MAPPING_REQUIRED")
	require.NotNil(t, first.Meta)
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR, first.Meta.GetRepoCpeMappingUpdatePath())
	assert.Empty(t, first.Meta.GetRepoCpeMappingHash())

	updated, syncMeta, err := syncOnce(t, client, mapping, handler.HandleConn)
	require.NoError(t, err)
	assert.True(t, updated)
	wantHash := cpemapping.HashMapping(mapping)
	assert.Equal(t, wantHash, syncMeta.GetRepoCpeMappingHash())

	cache.SetReport(&v4.IndexReport{HashId: "sync-integration-hash"}, nil, "")

	second, err := exchangeOnce(t, client, "", handler.HandleConn)
	require.NoError(t, err)
	require.NotNil(t, second.IndexReport)
	assert.Equal(t, "sync-integration-hash", second.IndexReport.GetHashId())
	assert.Equal(t, wantHash, second.Meta.GetRepoCpeMappingHash(), "the follow-up GetReport must reflect the mapping pushed via Sync")
}

// --- Compatibility persona helpers ---

// respondOnce reads one framed VMServiceRequest, builds a response, and writes
// it back. Used by fake agent personas that only differ in response content.
// Framing/protobuf failures here mean the harness itself is broken rather
// than the protocol under test, so they're logged (not asserted) to help
// triage without failing the test on the wrong line.
func respondOnce(t testing.TB, conn net.Conn, build func(*pb.VMServiceRequest) *pb.VMServiceResponse) {
	t.Helper()
	defer func() { _ = conn.Close() }()

	reqData, err := vsockframing.ReadFrame(conn, 1<<20)
	if err != nil {
		t.Logf("respondOnce: reading request frame: %v", err)
		return
	}
	var req pb.VMServiceRequest
	if err := proto.Unmarshal(reqData, &req); err != nil {
		t.Logf("respondOnce: unmarshalling request: %v", err)
		return
	}

	resp := build(&req)
	respData, err := proto.Marshal(resp)
	if err != nil {
		t.Logf("respondOnce: marshalling response: %v", err)
		return
	}
	if err := vsockframing.WriteFrame(conn, respData); err != nil {
		t.Logf("respondOnce: writing response frame: %v", err)
	}
}

// oldAgentResponder models a plausible older roxagent that:
//   - supports only get_report
//   - does not advertise supported_methods
//   - does not include facts, report_generated_at, or report_token
//   - always returns the full report (ignores last_known_token)
//   - returns UNKNOWN_METHOD for anything else
func oldAgentResponder(t testing.TB, report *v4.IndexReport) func(net.Conn) {
	return func(conn net.Conn) {
		respondOnce(t, conn, func(req *pb.VMServiceRequest) *pb.VMServiceResponse {
			resp := &pb.VMServiceResponse{
				Meta: &pb.ResponseMeta{
					AgentVersion: "roxagent-0.1.0",
				},
			}
			if req.GetGetReport() != nil {
				resp.Result = &pb.VMServiceResponse_GetReport{
					GetReport: &pb.GetReportResponse{IndexReport: report},
				}
			} else {
				resp.Result = &pb.VMServiceResponse_Error{
					Error: &pb.ErrorResponse{
						Code:    pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD,
						Message: "unsupported method",
					},
				}
			}
			return resp
		})
	}
}

// futureAgentResponder models a plausible future roxagent that:
//   - advertises extra supported_methods beyond get_report
//   - still handles get_report correctly
//   - includes richer metadata (facts, supported_methods)
//   - supports last_known_token (unchanged semantics)
//   - returns UNKNOWN_METHOD for methods it doesn't recognize
func futureAgentResponder(t testing.TB, report *v4.IndexReport, token string) func(net.Conn) {
	return func(conn net.Conn) {
		respondOnce(t, conn, func(req *pb.VMServiceRequest) *pb.VMServiceResponse {
			resp := &pb.VMServiceResponse{
				Meta: &pb.ResponseMeta{
					AgentVersion:     "roxagent-2.0.0-future",
					ReportToken:      token,
					SupportedMethods: []string{"get_report", "get_config", "submit_event"},
					Facts:            map[string]string{"os_id": "rhel", "protocol_version": "2"},
				},
			}
			switch {
			case req.GetGetReport() != nil:
				if req.GetGetReport().GetLastKnownToken() == token {
					resp.Result = &pb.VMServiceResponse_GetReport{
						GetReport: &pb.GetReportResponse{Unchanged: true},
					}
				} else {
					resp.Result = &pb.VMServiceResponse_GetReport{
						GetReport: &pb.GetReportResponse{IndexReport: report},
					}
				}
			default:
				resp.Result = &pb.VMServiceResponse_Error{
					Error: &pb.ErrorResponse{
						Code:    pb.ErrorCode_ERROR_CODE_UNKNOWN_METHOD,
						Message: "unsupported method",
					},
				}
			}
			return resp
		})
	}
}

// assertOldAgentReport checks the properties every old-agent compatibility
// case expects regardless of the request that triggered them: an old agent
// always serves the full report under its fixed identity and never
// advertises newer protocol metadata.
func assertOldAgentReport(t *testing.T, result *GetReportResult) {
	t.Helper()
	require.NotNil(t, result.IndexReport)
	assert.Equal(t, "compat-hash", result.IndexReport.GetHashId())
	assert.Equal(t, "roxagent-0.1.0", result.Meta.GetAgentVersion())
	assert.Empty(t, result.Meta.GetReportToken())
	assert.Empty(t, result.Meta.GetSupportedMethods())
	assert.Empty(t, result.Meta.GetFacts())
}

func TestGetReportCompatibility(t *testing.T) {
	report := &v4.IndexReport{HashId: "compat-hash"}
	client := NewClient([]string{CapabilityReportV1}, 10<<20)

	cases := map[string]struct {
		// Arguments
		makeResponderFunc func(t testing.TB) func(net.Conn)
		lastKnownToken    string
		// Expectations
		checkResult func(t *testing.T, result *GetReportResult)
	}{
		"old agent should serve get_report to current sensor": {
			makeResponderFunc: func(t testing.TB) func(net.Conn) { return oldAgentResponder(t, report) },
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				assertOldAgentReport(t, result)
			},
		},
		"old agent should always return full report ignoring last_known_token": {
			makeResponderFunc: func(t testing.TB) func(net.Conn) { return oldAgentResponder(t, report) },
			lastKnownToken:    "tok-1",
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged, "old agent does not support unchanged optimization")
				assertOldAgentReport(t, result)
			},
		},
		"future agent should serve get_report to current sensor": {
			makeResponderFunc: func(t testing.TB) func(net.Conn) { return futureAgentResponder(t, report, "tok-1") },
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "compat-hash", result.IndexReport.GetHashId())
				assert.Equal(t, "roxagent-2.0.0-future", result.Meta.GetAgentVersion())
				assert.Equal(t, "tok-1", result.Meta.GetReportToken())
				assert.Equal(t, []string{"get_report", "get_config", "submit_event"}, result.Meta.GetSupportedMethods())
				assert.Equal(t, "rhel", result.Meta.GetFacts()["os_id"])
				assert.Equal(t, "2", result.Meta.GetFacts()["protocol_version"])
			},
		},
		"future agent should return unchanged when last known token matches": {
			makeResponderFunc: func(t testing.TB) func(net.Conn) { return futureAgentResponder(t, report, "tok-3") },
			lastKnownToken:    "tok-3",
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.True(t, result.Unchanged)
				assert.Nil(t, result.IndexReport)
				assert.Equal(t, "tok-3", result.Meta.GetReportToken())
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := exchangeOnce(t, client, tc.lastKnownToken, tc.makeResponderFunc(t))

			require.NoError(t, err)
			if tc.checkResult != nil {
				tc.checkResult(t, result)
			}
		})
	}
}
