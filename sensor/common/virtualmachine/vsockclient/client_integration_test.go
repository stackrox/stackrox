package vsockclient

import (
	"net"
	"testing"

	roxagentvsock "github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
