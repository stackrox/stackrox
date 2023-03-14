package compliance

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/channelmultiplexer"
	"github.com/stackrox/rox/sensor/common"
)

// Multiplexer is a wrapper around pkg.channelmultiplexer that turns it into a sensor component.
// This is necessary since multiplexers are also used elsewhere, eg. compliance
type Multiplexer[T any] struct {
	mp channelmultiplexer.ChannelMultiplexer[T]
}

// NewMultiplexer creates a Multiplexer of type T, wrapped up as a sensor component
func NewMultiplexer[T any]() *Multiplexer[T] {
	multiplexer := Multiplexer[T]{
		mp: *channelmultiplexer.NewMultiplexer[T](),
	}
	return &multiplexer
}

// Notify is unimplemented, part of the component interface
func (c *Multiplexer[T]) Notify(e common.SensorComponentEvent) {
	// unimplemented
}

// Start starts the Multiplexer. It is important not to call this before all AddChannel calls
func (c *Multiplexer[T]) Start() error {
	// TODO maybe error if this fails(?)
	c.mp.Run()
	return nil
}

// Stop is unimplemented, part of the component interface
func (c *Multiplexer[T]) Stop(err error) {
}

// Capabilities is unimplemented, part of the component interface
func (c *Multiplexer[T]) Capabilities() []centralsensor.SensorCapability {
	return nil
}

// ProcessMessage is unimplemented, part of the component interface
func (c *Multiplexer[T]) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

// ResponsesC is unimplemented, part of the component interface
func (c *Multiplexer[T]) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

// AddChannel Adds a channel to ComplianceCommunicator, AddChannel must be called
// for ALL channels before calling Start()
func (c *Multiplexer[T]) AddChannel(channel <-chan *T) {
	c.mp.AddChannel(channel)
}

// GetCommandsC returns the multiplexed output channel combining all input channels added with AddChannel
func (c *Multiplexer[T]) GetCommandsC() <-chan *T {
	return c.mp.GetOutput()
}
