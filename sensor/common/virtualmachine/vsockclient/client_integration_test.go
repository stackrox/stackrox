package vsockclient

import (
	"net"
	"testing"

	roxagentvsock "github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// protocolHarness drives a real Sensor client against a real roxagent handler
// over net.Pipe(), validating the protocol contract end-to-end.
type protocolHarness struct {
	t       testing.TB
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

func newProtocolHarness(t testing.TB, opts protocolHarnessOptions) *protocolHarness {
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
		cache.SetReport(opts.seedReport, opts.seedFacts)
	}

	return &protocolHarness{
		t:       t,
		client:  NewClient(opts.capabilities, opts.maxResponseSize),
		cache:   cache,
		handler: roxagentvsock.NewHandler(cache, opts.agentVersion),
	}
}

func (h *protocolHarness) getReport(ifNewerThan uint32) (*GetReportResult, error) {
	h.t.Helper()

	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.handler.HandleConn(agentConn)
	}()

	result, err := h.client.GetReport(clientConn, ifNewerThan)
	<-done
	return result, err
}

func TestGetReportIntegration(t *testing.T) {
	cases := map[string]struct {
		// Arguments
		seedReport   *v4.IndexReport
		seedFacts    map[string]string
		agentVersion string
		ifNewerThan  uint32
		// Expectations
		wantErr     error
		checkResult func(t *testing.T, result *GetReportResult)
	}{
		"should return not ready when cache is empty": {
			wantErr: ErrNotReady,
		},
		"should return full report when cache has report": {
			seedReport: &v4.IndexReport{HashId: "integration-test-hash"},
			seedFacts:  map[string]string{"os": "rhel", "arch": "x86_64"},
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "integration-test-hash", result.IndexReport.GetHashId())

				require.NotNil(t, result.Meta)
				assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
				assert.NotNil(t, result.Meta.GetReportGeneratedAt())
				assert.Contains(t, result.Meta.GetSupportedMethods(), "get_report")
				assert.Equal(t, "rhel", result.Meta.GetFacts()["os"])
				assert.Equal(t, "x86_64", result.Meta.GetFacts()["arch"])
			},
		},
		"should return unchanged when generation matches": {
			seedReport:  &v4.IndexReport{HashId: "unchanged-hash"},
			ifNewerThan: 1,
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.True(t, result.Unchanged)
				assert.Nil(t, result.IndexReport)
				assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
			},
		},
		"should return full report when sensor generation exceeds agent after restart": {
			seedReport:  &v4.IndexReport{HashId: "post-restart-hash"},
			ifNewerThan: 5,
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged, "agent restarted (gen=1 < requested=5), must serve full report")
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "post-restart-hash", result.IndexReport.GetHashId())
				assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
			},
		},
		"should deliver all agent metadata to sensor": {
			seedReport:   &v4.IndexReport{HashId: "meta-hash"},
			seedFacts:    map[string]string{"os_id": "rhel", "kernel": "5.14.0", "instance_id": "i-abc123"},
			agentVersion: "roxagent-0.3.1-deadbeef",
			checkResult: func(t *testing.T, result *GetReportResult) {
				require.NotNil(t, result.Meta)
				assert.Equal(t, "roxagent-0.3.1-deadbeef", result.Meta.GetAgentVersion())
				assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
				assert.NotNil(t, result.Meta.GetReportGeneratedAt())
				assert.Contains(t, result.Meta.GetSupportedMethods(), "get_report")

				facts := result.Meta.GetFacts()
				assert.Equal(t, "rhel", facts["os_id"])
				assert.Equal(t, "5.14.0", facts["kernel"])
				assert.Equal(t, "i-abc123", facts["instance_id"])
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

			result, err := h.getReport(tc.ifNewerThan)

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

// --- Compatibility persona helpers ---

// exchangeWithResponder runs a single GetReport exchange using a fake agent
// responder instead of the real handler. This allows testing protocol
// compatibility with simulated older or future agent behavior.
func exchangeWithResponder(t *testing.T, client *Client, ifNewerThan uint32, responder func(net.Conn)) (*GetReportResult, error) {
	t.Helper()

	clientConn, agentConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		responder(agentConn)
	}()

	result, err := client.GetReport(clientConn, ifNewerThan)
	<-done
	return result, err
}

// oldAgentResponder models a plausible older roxagent that:
//   - supports only get_report
//   - does not advertise supported_methods
//   - does not include facts or report_generated_at
//   - always returns the full report (ignores if_newer_than_generation)
//   - returns UNKNOWN_METHOD for anything else
func oldAgentResponder(report *v4.IndexReport, generation uint32) func(net.Conn) {
	return func(conn net.Conn) {
		defer func() { _ = conn.Close() }()

		reqData, err := vsockframing.ReadFrame(conn, 1<<20)
		if err != nil {
			return
		}
		var req pb.VMServiceRequest
		if err := proto.Unmarshal(reqData, &req); err != nil {
			return
		}

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{
				AgentVersion:     "roxagent-0.1.0",
				ReportGeneration: generation,
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

		respData, err := proto.Marshal(resp)
		if err != nil {
			return
		}
		_ = vsockframing.WriteFrame(conn, respData)
	}
}

// futureAgentResponder models a plausible future roxagent that:
//   - advertises extra supported_methods beyond get_report
//   - still handles get_report correctly
//   - includes richer metadata (facts, supported_methods)
//   - supports if_newer_than_generation (unchanged semantics)
//   - returns UNKNOWN_METHOD for methods it doesn't recognize
func futureAgentResponder(report *v4.IndexReport, generation uint32) func(net.Conn) {
	return func(conn net.Conn) {
		defer func() { _ = conn.Close() }()

		reqData, err := vsockframing.ReadFrame(conn, 1<<20)
		if err != nil {
			return
		}
		var req pb.VMServiceRequest
		if err := proto.Unmarshal(reqData, &req); err != nil {
			return
		}

		resp := &pb.VMServiceResponse{
			Meta: &pb.ResponseMeta{
				AgentVersion:     "roxagent-2.0.0-future",
				ReportGeneration: generation,
				SupportedMethods: []string{"get_report", "get_config", "submit_event"},
				Facts:            map[string]string{"os_id": "rhel", "protocol_version": "2"},
			},
		}

		switch {
		case req.GetGetReport() != nil:
			if req.GetGetReport().GetIfNewerThanGeneration() == generation {
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

		respData, err := proto.Marshal(resp)
		if err != nil {
			return
		}
		_ = vsockframing.WriteFrame(conn, respData)
	}
}

func TestGetReportCompatibility(t *testing.T) {
	report := &v4.IndexReport{HashId: "compat-hash"}
	client := NewClient([]string{CapabilityReportV1}, 10<<20)

	cases := map[string]struct {
		// Arguments
		responder   func(net.Conn)
		ifNewerThan uint32
		// Expectations
		checkResult func(t *testing.T, result *GetReportResult)
	}{
		"old agent should serve get_report to current sensor": {
			responder: oldAgentResponder(report, 1),
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "compat-hash", result.IndexReport.GetHashId())
				assert.Equal(t, "roxagent-0.1.0", result.Meta.GetAgentVersion())
				assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
				assert.Empty(t, result.Meta.GetSupportedMethods())
				assert.Empty(t, result.Meta.GetFacts())
			},
		},
		"old agent should always return full report ignoring if_newer_than": {
			responder:   oldAgentResponder(report, 1),
			ifNewerThan: 1,
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged, "old agent does not support unchanged optimization")
				require.NotNil(t, result.IndexReport)
			},
		},
		"future agent should serve get_report to current sensor": {
			responder: futureAgentResponder(report, 1),
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.False(t, result.Unchanged)
				require.NotNil(t, result.IndexReport)
				assert.Equal(t, "compat-hash", result.IndexReport.GetHashId())
				assert.Equal(t, "roxagent-2.0.0-future", result.Meta.GetAgentVersion())
				assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
				assert.Equal(t, []string{"get_report", "get_config", "submit_event"}, result.Meta.GetSupportedMethods())
				assert.Equal(t, "rhel", result.Meta.GetFacts()["os_id"])
				assert.Equal(t, "2", result.Meta.GetFacts()["protocol_version"])
			},
		},
		"future agent should return unchanged when generation matches": {
			responder:   futureAgentResponder(report, 3),
			ifNewerThan: 3,
			checkResult: func(t *testing.T, result *GetReportResult) {
				assert.True(t, result.Unchanged)
				assert.Nil(t, result.IndexReport)
				assert.Equal(t, uint32(3), result.Meta.GetReportGeneration())
			},
		},
		"future agent extra supported_methods are discoverable by sensor": {
			responder: futureAgentResponder(report, 1),
			checkResult: func(t *testing.T, result *GetReportResult) {
				methods := result.Meta.GetSupportedMethods()
				assert.Contains(t, methods, "get_report")
				assert.Contains(t, methods, "get_config")
				assert.Contains(t, methods, "submit_event")
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := exchangeWithResponder(t, client, tc.ifNewerThan, tc.responder)

			require.NoError(t, err)
			if tc.checkResult != nil {
				tc.checkResult(t, result)
			}
		})
	}
}
