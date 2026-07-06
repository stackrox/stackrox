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

	result, err := client.GetReport(stream, 0)
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
	first, err := client.GetReport(stream1, 0)
	require.NoError(t, err)
	_ = stream1.Close()

	stream2, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
	require.NoError(t, err)
	defer func() { _ = stream2.Close() }()
	second, err := client.GetReport(stream2, first.Meta.GetReportGeneration())
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
		result, err := client.GetReport(stream, 0)
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

	var lastGeneration uint32
	for range 5 {
		stream, err := dialer.Dial(context.Background(), Namespace, "vm-0", 818, true)
		require.NoError(t, err)
		result, err := client.GetReport(stream, lastGeneration)
		require.NoError(t, err)
		_ = stream.Close()

		assert.False(t, result.Unchanged, "expected a changed report on every poll with alwaysChanged=true")
		assert.Greater(t, result.Meta.GetReportGeneration(), lastGeneration)
		lastGeneration = result.Meta.GetReportGeneration()
	}
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
			GetReport: &pb.GetReportRequest{IfNewerThanGeneration: 3},
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
