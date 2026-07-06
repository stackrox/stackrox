package vsockserver

import (
	"testing"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeReportProvider_GetReport_SizeSelection(t *testing.T) {
	tests := map[string]struct {
		facts        map[string]string
		wantPackages int
	}{
		"small size fact selects the small canned report": {
			facts:        map[string]string{"report_size": "small"},
			wantPackages: 50,
		},
		"medium size fact selects the medium canned report": {
			facts:        map[string]string{"report_size": "medium"},
			wantPackages: 524,
		},
		"large size fact selects the large canned report": {
			facts:        map[string]string{"report_size": "large"},
			wantPackages: 2000,
		},
		"missing report_size fact defaults to medium": {
			facts:        nil,
			wantPackages: 524,
		},
		"unrecognized report_size fact defaults to medium": {
			facts:        map[string]string{"report_size": "extra-large"},
			wantPackages: 524,
		},
		"nil meta defaults to medium": {
			wantPackages: 524,
		},
	}

	provider := NewFakeReportProvider()
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var meta *pb.RequestMeta
			if tt.facts != nil {
				meta = &pb.RequestMeta{Facts: tt.facts}
			}

			report, facts, generation, epoch, generatedAt, ready := provider.GetReport(nil, meta)

			assert.True(t, ready)
			require.NotNil(t, report)
			assert.Len(t, report.GetContents().GetPackages(), tt.wantPackages)
			assert.Equal(t, uint32(1), generation, "canned reports are built once, generation should always be 1")
			assert.NotZero(t, epoch)
			assert.Equal(t, "RHEL", facts["detected_os"])
			assert.False(t, generatedAt.IsZero())
		})
	}
}

func TestFakeReportProvider_GetReport_EpochRandomizedPerResponse(t *testing.T) {
	provider := NewFakeReportProvider()
	meta := &pb.RequestMeta{Facts: map[string]string{"report_size": "small"}}

	seen := make(map[uint32]bool)
	const iterations = 20
	for range iterations {
		_, _, generation, epoch, _, ready := provider.GetReport(nil, meta)
		assert.True(t, ready)
		assert.Equal(t, uint32(1), generation, "generation must stay fixed while epoch churns")
		assert.NotZero(t, epoch, "0 is reserved to mean \"agent predates this field\"")
		seen[epoch] = true
	}
	// Not a strict guarantee (randomness could theoretically repeat), but with
	// 20 draws from a uint32 space, collisions overwhelmingly indicate a bug
	// (e.g. epoch not actually being re-randomized) rather than bad luck.
	assert.Greater(t, len(seen), iterations/2, "epoch should differ across responses, not repeat a fixed value")
}

func TestFakeReportProvider_GetReport_ReportsIdenticalAcrossCalls(t *testing.T) {
	// Canned reports are pre-built once at construction -- unlike epoch,
	// report content must not change between calls.
	provider := NewFakeReportProvider()
	meta := &pb.RequestMeta{Facts: map[string]string{"report_size": "medium"}}

	report1, _, _, _, _, _ := provider.GetReport(nil, meta)
	report2, _, _, _, _, _ := provider.GetReport(nil, meta)

	assert.Same(t, report1, report2, "the same canned report instance should be reused across calls")
}

func TestNewHandler_WithFakeReportProvider_ServesCannedReportBySize(t *testing.T) {
	handler := NewHandler(NewFakeReportProvider(), "loadtest-agent")

	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{
			RequestId: "req-loadtest",
			Facts:     map[string]string{"report_size": "large"},
		},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{LastKnownGeneration: 0}},
	}

	resp := sendAndReceive(t, handler, req)

	require.NotNil(t, resp.GetGetReport())
	assert.False(t, resp.GetGetReport().GetUnchanged())
	assert.Len(t, resp.GetGetReport().GetIndexReport().GetContents().GetPackages(), 2000)
	assert.NotZero(t, resp.GetMeta().GetEpoch())
	assert.Equal(t, uint32(1), resp.GetMeta().GetReportGeneration())
}

func TestNewHandler_WithFakeReportProvider_EpochMismatchForcesFullReportEvenWhenGenerationUnchanged(t *testing.T) {
	// Mirrors what real Sensor's VMScraper relies on (ROX-35597): even
	// though the fake provider's generation never changes, its epoch differs
	// on every response, so a caller polling "if_newer_than == last known
	// generation" still sees a mismatched epoch and knows to treat it as
	// changed (Sensor-side behavior tested separately in vmscraper).
	handler := NewHandler(NewFakeReportProvider(), "loadtest-agent")
	req := &pb.VMServiceRequest{
		Meta:   &pb.RequestMeta{RequestId: "req-1"},
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{LastKnownGeneration: 1}},
	}

	resp1 := sendAndReceive(t, handler, req)
	resp2 := sendAndReceive(t, handler, req)

	require.NotNil(t, resp1.GetGetReport())
	require.NotNil(t, resp2.GetGetReport())
	assert.True(t, resp1.GetGetReport().GetUnchanged(), "generation matches, roxagent reports unchanged")
	assert.True(t, resp2.GetGetReport().GetUnchanged())
	assert.NotEqual(t, resp1.GetMeta().GetEpoch(), resp2.GetMeta().GetEpoch(),
		"epoch must still churn even on the unchanged path, so Sensor's epoch-mismatch check can force a full report")
}
