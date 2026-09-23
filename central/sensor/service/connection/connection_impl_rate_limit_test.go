package connection

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/rate"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRateLimiter struct {
	allowed bool
	reason  string
}

func (s stubRateLimiter) TryConsume(string, *central.MsgFromSensor) (bool, string) {
	return s.allowed, s.reason
}

type recordingAdminStream struct {
	produced []*events.AdministrationEvent
}

func (s *recordingAdminStream) Consume(_ concurrency.Waitable) func(yield func(*events.AdministrationEvent) bool) {
	return func(func(*events.AdministrationEvent) bool) {}
}

func (s *recordingAdminStream) Produce(event *events.AdministrationEvent) {
	s.produced = append(s.produced, event)
}

func TestMultiplexedPush_RateLimitedNodeIndexReportDropsWithoutNACK(t *testing.T) {
	admin := &recordingAdminStream{}
	conn := newRateLimitedTestConnection(stubRateLimiter{
		allowed: false,
		reason:  rate.ReasonRateLimitExceeded,
	}, admin)

	conn.multiplexedPush(t.Context(), nodeIndexMsg("node-abc"), nil)

	assert.Empty(t, conn.sendC, "node index reports must not receive a rate-limit NACK")
	require.Len(t, admin.produced, 1)
	assert.Contains(t, admin.produced[0].Message, "Node index reports")
	assert.Contains(t, admin.produced[0].Hint, env.NodeIndexReportRateLimit.EnvVar())
}

func TestMultiplexedPush_RateLimitedVMIndexReportSendsNACK(t *testing.T) {
	admin := &recordingAdminStream{}
	conn := newRateLimitedTestConnection(stubRateLimiter{
		allowed: false,
		reason:  rate.ReasonRateLimitExceeded,
	}, admin)

	conn.multiplexedPush(t.Context(), vmIndexMsg("vm-1"), nil)

	nack := receiveSensorACK(t, conn)
	assert.Equal(t, central.SensorACK_NACK, nack.GetAction())
	assert.Equal(t, central.SensorACK_VM_INDEX_REPORT, nack.GetMessageType())
	assert.Equal(t, "vm-1:100", nack.GetResourceId())
	assert.Equal(t, centralsensor.SensorACKReasonRateLimited, nack.GetReason())

	require.Len(t, admin.produced, 1)
	assert.Contains(t, admin.produced[0].Message, "VM index reports")
}

func TestMultiplexedPush_RateLimitedNodeInventoryDoesNotNACK(t *testing.T) {
	admin := &recordingAdminStream{}
	conn := newRateLimitedTestConnection(stubRateLimiter{
		allowed: false,
		reason:  rate.ReasonRateLimitExceeded,
	}, admin)

	conn.multiplexedPush(t.Context(), nodeInventoryMsg("node-1"), nil)

	assert.Empty(t, conn.sendC, "node inventory must not receive a rate-limit NACK")
	assert.Empty(t, admin.produced, "node inventory must not emit a rate-limit admin event")
}

func newRateLimitedTestConnection(rl rateLimiter, admin events.Stream) *sensorConnection {
	return &sensorConnection{
		clusterID:         "cluster-1",
		rl:                rl,
		capabilities:      set.NewSet(centralsensor.SensorACKSupport),
		sendC:             make(chan *central.MsgToSensor, 1),
		stopSig:           concurrency.NewErrorSignal(),
		adminEventsStream: admin,
	}
}

func receiveSensorACK(t *testing.T, conn *sensorConnection) *central.SensorACK {
	t.Helper()
	select {
	case msg := <-conn.sendC:
		ack := msg.GetSensorAck()
		require.NotNil(t, ack)
		return ack
	default:
		t.Fatal("expected a SensorACK on sendC")
		return nil
	}
}
