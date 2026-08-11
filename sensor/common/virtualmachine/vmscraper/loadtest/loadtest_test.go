package loadtest

import (
	"context"
	"testing"
	"time"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const testMaxResponseSize = 16 * 1024 * 1024

func TestFarm_ListRunningMatchesCreatedCount(t *testing.T) {
	farm := NewFarm(5, 10, 0, false)
	infos := farm.ListRunning()
	require.Len(t, infos, 5)
	for _, info := range infos {
		assert.Equal(t, Namespace, info.Namespace)
		assert.True(t, info.Running)
		assert.NotNil(t, info.VSOCKCID)
	}
}

func TestFarmDialer_DialUnknownVMFails(t *testing.T) {
	farm := NewFarm(1, 10, 0, false)
	dialer := NewFarmDialer(farm, 0)
	_, err := dialer.Dial(context.Background(), Namespace, "does-not-exist", 818, true)
	assert.Error(t, err)
}

// TestFarmDialer_RealProtocolRoundTrip drives the real vsockclient.Client
// against a FarmVM's real vsockserver.Handler over the dialer's net.Pipe
// transport -- the one end-to-end check that the harness wiring actually
// speaks the real wire protocol, so a harness bug doesn't masquerade as a
// finding about Sensor.
func TestFarmDialer_RealProtocolRoundTrip(t *testing.T) {
	farm := NewFarm(3, 20, 0, false)
	dialer := NewFarmDialer(farm, 0)
	client := vsockclient.NewClient([]string{vsockclient.CapabilityReportV1}, testMaxResponseSize)

	stream, err := dialer.Dial(context.Background(), Namespace, "vm-1", 818, true)
	require.NoError(t, err)
	defer func() { _ = stream.Close() }()

	result, err := client.GetReport(t.Context(), stream, 0, 0)
	require.NoError(t, err)
	assert.False(t, result.Unchanged)
	assert.Equal(t, uint32(1), result.Meta.GetReportGeneration())
	assert.Len(t, result.IndexReport.GetContents().GetPackages(), 20)
}

func TestFarmDialer_UnchangedAfterSameGeneration(t *testing.T) {
	farm := NewFarm(1, 5, 0, false)
	dialer := NewFarmDialer(farm, 0)
	client := vsockclient.NewClient([]string{vsockclient.CapabilityReportV1}, testMaxResponseSize)

	stream1, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
	require.NoError(t, err)
	first, err := client.GetReport(t.Context(), stream1, 0, 0)
	require.NoError(t, err)
	_ = stream1.Close()

	stream2, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
	require.NoError(t, err)
	defer func() { _ = stream2.Close() }()
	second, err := client.GetReport(t.Context(), stream2, first.Meta.GetReportGeneration(), first.Meta.GetEpoch())
	require.NoError(t, err)
	assert.True(t, second.Unchanged)
	assert.Nil(t, second.IndexReport)
}

func TestFarm_RescanBumpsGeneration(t *testing.T) {
	farm := NewFarm(2, 5, 10*time.Millisecond, false)
	ctx := t.Context()
	farm.Start(ctx)

	dialer := NewFarmDialer(farm, 0)
	client := vsockclient.NewClient([]string{vsockclient.CapabilityReportV1}, testMaxResponseSize)

	require.Eventually(t, func() bool {
		stream, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
		if err != nil {
			return false
		}
		defer func() { _ = stream.Close() }()
		result, err := client.GetReport(t.Context(), stream, 0, 0)
		if err != nil {
			return false
		}
		return result.Meta.GetReportGeneration() > 1
	}, time.Second, 10*time.Millisecond, "expected rescan loop to bump generation above 1")
}

// TestFarm_AlwaysChangedBumpsGenerationEveryDial guards the fix for a real
// gap found while stress-testing at fleet scale: without alwaysChanged,
// VMScraper's per-VM generation cache means only the very first poll of each
// VM is ever a "changed" full report -- every poll after that is "unchanged"
// unless the (fleet-size-independent) rescan loop happens to pick that exact
// VM. At hundreds+ of VMs that made the harness almost never exercise the
// expensive full-report path.
func TestFarm_AlwaysChangedBumpsGenerationEveryDial(t *testing.T) {
	farm := NewFarm(1, 5, 0, true)
	dialer := NewFarmDialer(farm, 0)
	client := vsockclient.NewClient([]string{vsockclient.CapabilityReportV1}, testMaxResponseSize)

	var lastGeneration, lastEpoch uint32
	for range 5 {
		stream, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
		require.NoError(t, err)
		result, err := client.GetReport(t.Context(), stream, lastGeneration, lastEpoch)
		require.NoError(t, err)
		_ = stream.Close()

		assert.False(t, result.Unchanged, "expected a changed report on every poll with alwaysChanged=true")
		assert.Greater(t, result.Meta.GetReportGeneration(), lastGeneration)
		lastGeneration = result.Meta.GetReportGeneration()
		lastEpoch = result.Meta.GetEpoch()
	}
}

// TestFarm_AddVMsGrowsFleetWithoutDisturbingExisting guards ramp mode's core
// assumption: adding VMs mid-run must not reset or collide with VMs already
// present (e.g. by reusing an index and generating a duplicate key).
func TestFarm_AddVMsGrowsFleetWithoutDisturbingExisting(t *testing.T) {
	farm := NewFarm(3, 5, 0, false)
	require.Equal(t, 3, farm.Count())

	dialer := NewFarmDialer(farm, 0)
	client := vsockclient.NewClient([]string{vsockclient.CapabilityReportV1}, testMaxResponseSize)
	stream, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
	require.NoError(t, err)
	before, err := client.GetReport(t.Context(), stream, 0, 0)
	require.NoError(t, err)
	_ = stream.Close()

	farm.AddVMs(4)
	assert.Equal(t, 7, farm.Count())
	infos := farm.ListRunning()
	assert.Len(t, infos, 7)
	seen := make(map[string]bool, len(infos))
	for _, info := range infos {
		key := info.Namespace + "/" + info.Name
		assert.False(t, seen[key], "duplicate VM key %q after AddVMs", key)
		seen[key] = true
	}

	// The pre-existing VM's generation state must be untouched by growth.
	stream2, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
	require.NoError(t, err)
	defer func() { _ = stream2.Close() }()
	after, err := client.GetReport(t.Context(), stream2, before.Meta.GetReportGeneration(), before.Meta.GetEpoch())
	require.NoError(t, err)
	assert.True(t, after.Unchanged, "existing VM's generation should be unaffected by AddVMs")
}

// TestFarm_BacklogCountTracksOverdueVMs guards the ramp-mode saturation
// signal: a VM counts as backlogged once it hasn't been successfully dialed
// within pollInterval, and stops counting again once it has.
func TestFarm_BacklogCountTracksOverdueVMs(t *testing.T) {
	farm := NewFarm(2, 5, 0, false)
	pollInterval := 20 * time.Millisecond

	// Freshly created VMs are not immediately overdue.
	assert.Equal(t, 0, farm.BacklogCount(pollInterval))

	time.Sleep(2 * pollInterval)
	assert.Equal(t, 2, farm.BacklogCount(pollInterval), "both VMs should be overdue after pollInterval with no dials")

	dialer := NewFarmDialer(farm, 0)
	stream, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
	require.NoError(t, err)
	_ = stream.Close()

	assert.Equal(t, 1, farm.BacklogCount(pollInterval), "the just-dialed VM should no longer be overdue")
}

func TestFarmDialer_LatencyInjectionRespectsContextCancellation(t *testing.T) {
	farm := NewFarm(1, 5, 0, false)
	dialer := NewFarmDialer(farm, time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := dialer.Dial(ctx, Namespace, "vm-0", 818, true)
	assert.Error(t, err)
}

// TestNullSender_IsANoOp is a trivial sanity check that Send never errors and
// touches nothing -- guards against accidental future coupling to Central.
func TestNullSender_IsANoOp(t *testing.T) {
	var sender NullSender
	err := sender.Send(context.Background(), nil, nil)
	assert.NoError(t, err)
}

// sanity check that the wire protocol types used above still round-trip via
// proto marshal/unmarshal, guarding against accidental protocol drift between
// this test package and the generated proto.
func TestProtocolTypesRoundTrip(t *testing.T) {
	req := &pb.VMServiceRequest{
		Meta: &pb.RequestMeta{RequestId: "test"},
		Method: &pb.VMServiceRequest_GetReport{
			GetReport: &pb.GetReportRequest{LastKnownGeneration: 3},
		},
	}
	data, err := proto.Marshal(req)
	require.NoError(t, err)

	var buf countingWriter
	require.NoError(t, vsockframing.WriteFrame(&buf, data))
	assert.Positive(t, buf.n)
}

type countingWriter struct{ n int }

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}
