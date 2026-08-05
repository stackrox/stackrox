package compliance

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/goleak"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	pubsubDispatcher "github.com/stackrox/rox/sensor/common/pubsub/dispatcher"
	"github.com/stackrox/rox/sensor/common/pubsub/lane"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestMultiplexer(t *testing.T) {
	suite.Run(t, new(MultiplexerTestSuite))
}

type MultiplexerTestSuite struct {
	suite.Suite
}

func (s *MultiplexerTestSuite) TearDownTest() {
	goleak.AssertNoGoroutineLeaks(s.T())
}

// newTestMultiplexer builds a Multiplexer whose underlying channelmultiplexer.ChannelMultiplexer
// is bound to a cancellable context, so the test can fully unwind FanIn's goroutines on cleanup --
// production code has no such lifecycle hook today (Multiplexer.Stop() does not tear down the
// underlying fan-in), so without this the legacy channels/pubSubIn would leak goroutines forever.
func newTestMultiplexer(dispatcher common.PubSubDispatcher) (*Multiplexer, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	mp := &Multiplexer{
		mp:               *channelmultiplexer.NewMultiplexer[common.MessageToComplianceWithAddress](channelmultiplexer.WithContext[common.MessageToComplianceWithAddress](ctx)),
		components:       []common.ComplianceComponent{},
		stopper:          concurrency.NewStopper(),
		pubSubDispatcher: dispatcher,
		pubSubIn:         make(chan common.MessageToComplianceWithAddress),
	}
	return mp, cancel
}

// legacyComplianceComponent simulates a producer still on the raw ComplianceC() channel,
// eg. the VM Handler, which stays on that path until ROX-36011 migrates it.
type legacyComplianceComponent struct {
	unimplemented.Receiver
	toCompliance chan common.MessageToComplianceWithAddress
	stopper      concurrency.Stopper
}

func newLegacyComplianceComponent() *legacyComplianceComponent {
	return &legacyComplianceComponent{
		toCompliance: make(chan common.MessageToComplianceWithAddress),
		stopper:      concurrency.NewStopper(),
	}
}

func (c *legacyComplianceComponent) Name() string                                   { return "legacyComplianceComponent" }
func (c *legacyComplianceComponent) Start() error                                   { return nil }
func (c *legacyComplianceComponent) Stop()                                          {}
func (c *legacyComplianceComponent) Capabilities() []centralsensor.SensorCapability { return nil }
func (c *legacyComplianceComponent) ResponsesC() <-chan *message.ExpiringMessage    { return nil }
func (c *legacyComplianceComponent) Notify(_ common.SensorComponentEvent)           {}
func (c *legacyComplianceComponent) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}
func (c *legacyComplianceComponent) ComplianceC() <-chan common.MessageToComplianceWithAddress {
	return c.toCompliance
}

func newTestMultiplexerDispatcher(t *testing.T) common.PubSubDispatcher {
	t.Helper()
	dispatcher, err := pubsubDispatcher.NewDispatcher(pubsubDispatcher.WithLaneConfigs(
		[]pubsub.LaneConfig{
			lane.NewBlockingLane(pubsub.ComplianceAckLane),
		},
	))
	require.NoError(t, err)
	return dispatcher
}

// TestPubSubAndLegacyProducersBothFeedComplianceC verifies the Multiplexer accepts a
// PubSub-published ComplianceAckEvent (eg. from the Node Inventory Handler) and a legacy
// ComplianceC() channel (eg. from the still-unmigrated VM Handler) at the same time, folding
// both into one output stream -- the dual-consumer bridge state ROX-35978 introduces ahead of
// ROX-36011 migrating the VM Handler leg.
func (s *MultiplexerTestSuite) TestPubSubAndLegacyProducersBothFeedComplianceC() {
	t := s.T()
	t.Setenv(features.SensorInternalPubSub.EnvVar(), "true")

	dispatcher := newTestMultiplexerDispatcher(t)
	defer dispatcher.Stop()

	legacy := newLegacyComplianceComponent()
	mp, cancel := newTestMultiplexer(dispatcher)
	defer cancel()
	mp.AddComponentWithComplianceC(legacy)
	s.Require().NoError(mp.Start())
	defer mp.Stop()

	require.NoError(t, dispatcher.Publish(&ComplianceAckEvent{Msg: common.MessageToComplianceWithAddress{Hostname: "node-pubsub"}}))
	go func() {
		legacy.toCompliance <- common.MessageToComplianceWithAddress{Hostname: "node-legacy"}
	}()

	seen := map[string]bool{}
	for range 2 {
		select {
		case msg := <-mp.ComplianceC():
			seen[msg.Hostname] = true
		case <-time.After(2 * time.Second):
			s.FailNow("timed out waiting for messages on ComplianceC()")
		}
	}
	s.True(seen["node-pubsub"], "expected the PubSub-sourced message to appear on ComplianceC()")
	s.True(seen["node-legacy"], "expected the legacy channel-sourced message to appear on ComplianceC()")
}

// TestPubSubDisabledFallsBackToLegacyOnly verifies that with PubSub disabled, the Multiplexer
// behaves exactly as before: only legacy ComplianceC() channels are fanned in.
func (s *MultiplexerTestSuite) TestPubSubDisabledFallsBackToLegacyOnly() {
	legacy := newLegacyComplianceComponent()
	mp, cancel := newTestMultiplexer(nil)
	defer cancel()
	mp.AddComponentWithComplianceC(legacy)
	s.Require().NoError(mp.Start())
	defer mp.Stop()

	go func() {
		legacy.toCompliance <- common.MessageToComplianceWithAddress{Hostname: "node-legacy"}
	}()

	select {
	case msg := <-mp.ComplianceC():
		s.Equal("node-legacy", msg.Hostname)
	case <-time.After(2 * time.Second):
		s.FailNow("timed out waiting for legacy message on ComplianceC()")
	}
}
