package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/sensor/common"
)

// Multiplexer is a wrapper around pkg.channelmultiplexer that turns it into a sensor component.
// This is necessary since multiplexers are also used elsewhere, eg. compliance
type Multiplexer struct {
	mp         channelmultiplexer.ChannelMultiplexer[common.MessageToComplianceWithAddress]
	components []common.ComplianceComponent
}

// NewMultiplexer creates a Multiplexer of type T, wrapped up as a sensor component
func NewMultiplexer() *Multiplexer {
	multiplexer := Multiplexer{
		mp:         *channelmultiplexer.NewMultiplexer[common.MessageToComplianceWithAddress](),
		components: []common.ComplianceComponent{},
	}
	return &multiplexer
}

// Notify is unimplemented, part of the component interface
func (c *Multiplexer) Notify(_ common.SensorComponentEvent) {
	// unimplemented
}

// Start starts the Multiplexer. It is important not to call this before all addChannel calls
func (c *Multiplexer) Start() error {
	log.Debugf("Multiplexer starts")
	return c.run()
}

func (c *Multiplexer) run() error {
	// Multiplexer must start after all components from the c.components. Otherwise, comp.ComplianceC may be nil
	for _, comp := range c.components {
		c.addChannel(comp.ComplianceC())
	}
	c.mp.Run()
	return nil
}

// Stop is unimplemented, part of the component interface
func (c *Multiplexer) Stop(_ error) {
}

// Capabilities is unimplemented, part of the component interface
func (c *Multiplexer) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage is unimplemented, part of the component interface
func (c *Multiplexer) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

// ResponsesC is unimplemented, part of the component interface
func (c *Multiplexer) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

// AddComponentWithComplianceC registers components that will use the .ComplianceC() for communicating with Compliance
func (c *Multiplexer) AddComponentWithComplianceC(comp ...common.ComplianceComponent) {
	c.components = append(c.components, comp...)
}

// addChannel Adds a channel to ComplianceCommunicator, addChannel must be called
// for ALL channels before calling Start()
func (c *Multiplexer) addChannel(channel <-chan *common.MessageToComplianceWithAddress) {
	if channel == nil {
		panic("addChannel cannot work with nil channels")
	}
	c.mp.AddChannel(channel)
}

// GetCommandsC returns the multiplexed output channel combining all input channels added with addChannel
func (c *Multiplexer) GetCommandsC() <-chan *common.MessageToComplianceWithAddress {
	return c.mp.GetOutput()
}
