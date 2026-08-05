package compliance

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/pubsub"
	"github.com/stackrox/rox/sensor/common/unimplemented"
)

var _ common.ComplianceComponent = (*Multiplexer)(nil)

// Multiplexer is a wrapper around pkg.channelmultiplexer that turns it into a sensor component.
// This is necessary since multiplexers are also used elsewhere, eg. compliance
type Multiplexer struct {
	unimplemented.Receiver

	mp               channelmultiplexer.ChannelMultiplexer[common.MessageToComplianceWithAddress]
	components       []common.ComplianceComponent
	stopper          concurrency.Stopper
	pubSubDispatcher common.PubSubDispatcher
	// pubSubIn receives ComplianceAckEvents published by producers migrated to PubSub (eg. the
	// Node Inventory Handler); it is folded into the same fan-in mix as any legacy ComplianceC()
	// channel, so producers still on raw channels (eg. the VM Handler) keep working unchanged.
	pubSubIn chan common.MessageToComplianceWithAddress
}

func (c *Multiplexer) Name() string {
	return "compliance.Multiplexer"
}

// Stopped returns a signal allowing to check whether the component has been stopped
func (c *Multiplexer) Stopped() concurrency.ReadOnlyErrorSignal {
	return c.stopper.Client().Stopped()
}

// NewMultiplexer creates a Multiplexer of type T, wrapped up as a sensor component
func NewMultiplexer(pubSubDispatcher common.PubSubDispatcher) *Multiplexer {
	multiplexer := Multiplexer{
		mp:               *channelmultiplexer.NewMultiplexer[common.MessageToComplianceWithAddress](),
		components:       []common.ComplianceComponent{},
		stopper:          concurrency.NewStopper(),
		pubSubDispatcher: pubSubDispatcher,
		pubSubIn:         make(chan common.MessageToComplianceWithAddress),
	}
	return &multiplexer
}

// Notify is unimplemented, part of the component interface
func (c *Multiplexer) Notify(_ common.SensorComponentEvent) {
	// unimplemented
}

// Start starts the Multiplexer. It is important not to call this before all addChannel calls
func (c *Multiplexer) Start() error {
	return c.run()
}

func (c *Multiplexer) run() error {
	// Multiplexer must start after all components from the c.components. Otherwise, comp.ComplianceC may be nil
	for _, comp := range c.components {
		c.addChannel(comp.ComplianceC())
	}
	if features.SensorInternalPubSub.Enabled() && c.pubSubDispatcher != nil {
		if err := c.pubSubDispatcher.RegisterConsumerToLane(
			pubsub.ComplianceMultiplexerAckConsumer,
			pubsub.ComplianceAckTopic,
			pubsub.ComplianceAckLane,
			c.handleComplianceAckEvent,
		); err != nil {
			return errors.Wrap(err, "failed to register compliance multiplexer ack consumer")
		}
		c.addChannel(c.pubSubIn)
	}
	c.mp.Run()
	return nil
}

// handleComplianceAckEvent bridges a PubSub-published ComplianceAckEvent into the legacy
// channel-based fan-in, so producers still on raw channels and producers already migrated to
// PubSub feed the same ComplianceC() output stream.
func (c *Multiplexer) handleComplianceAckEvent(event pubsub.Event) error {
	ackEvent, ok := event.(*ComplianceAckEvent)
	if !ok {
		return errors.Errorf("unexpected event type: %T", event)
	}
	select {
	case <-c.stopper.Client().Stopped().Done():
	case c.pubSubIn <- ackEvent.Msg:
	}
	return nil
}

// Stop stops the component
func (c *Multiplexer) Stop() {
	c.stopper.Client().Stop()
}

// Capabilities is unimplemented, part of the component interface
func (c *Multiplexer) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage is unimplemented, part of the component interface

// ResponsesC is unimplemented, part of the component interface
func (c *Multiplexer) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

// AddComponentWithComplianceC registers components that will use the .ComplianceC() for communicating with Compliance
func (c *Multiplexer) AddComponentWithComplianceC(comp ...common.ComplianceComponent) {
	c.components = append(c.components, comp...)
}

// addChannel Adds a channel to ComplianceCommunicator, addChannel must be called
// for ALL channels before calling Start()
func (c *Multiplexer) addChannel(channel <-chan common.MessageToComplianceWithAddress) {
	if channel == nil {
		utils.Must(errors.New("Multiplexer.AddChannel() cannot work with nil channels"))
	}
	c.mp.AddChannel(channel)
}

// ComplianceC returns the multiplexed output channel combining all input channels added with addChannel
func (c *Multiplexer) ComplianceC() <-chan common.MessageToComplianceWithAddress {
	return c.mp.GetOutput()
}
