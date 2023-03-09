package compliance

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
)

// MessageToComplianceWithAddress adds the hostname to sensor.MsgToCompliance so we know where to send it to.
type MessageToComplianceWithAddress struct {
	msg       *sensor.MsgToCompliance
	hostname  string
	broadcast bool
}

// Multiplexer combines n input channels of type T into one output channel of type T
type Multiplexer[T any] struct {
	inputChannels  []<-chan *T
	outputCommands chan *T
	// connectionMap  map[string]sensor.ComplianceService_CommunicateServer
	// manager connectionManager

	wg      sync.WaitGroup
	started concurrency.Signal
}

// Notify is unimplemented, part of the component interface
func (c *Multiplexer[T]) Notify(e common.SensorComponentEvent) {
	// unimplemented
}

// Start starts the Multiplexer. It is important not to call this before all AddChannel calls
func (c *Multiplexer[T]) Start() error {
	// TODO maybe error if this fails(?)
	c.run()
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

// NewMultiplexer creates a Multiplexer of type T
func NewMultiplexer[T any]() *Multiplexer[T] {
	communicator := Multiplexer[T]{
		inputChannels:  make([]<-chan *T, 0),
		outputCommands: make(chan *T),
		started:        concurrency.Signal{}}

	return &communicator
}

// AddChannel Adds a channel to ComplianceCommunicator, AddChannel must be called
// for ALL channels before calling Start()
func (c *Multiplexer[T]) AddChannel(channel <-chan *T) {
	if c.started.IsDone() {
		panic("Cannot AddChannel after component is started")
	}
	c.inputChannels = append(c.inputChannels, channel)
}

func (c *Multiplexer[T]) run() {
	c.started.Signal()
	ctx := context.Background()

	output := FanIn[T](ctx, c.inputChannels...)
	for o := range output {
		c.outputCommands <- o
	}
}

// FanIn multiplexes multiple input channels into one output channel and
// finishes when all input channels are closed
func FanIn[T any](ctx context.Context, channels ...<-chan *T) <-chan *T {
	multiplexedStream := make(chan *T)
	wg := sync.WaitGroup{}

	multiplex := func(ch <-chan *T) {
		defer wg.Done()
		for i := range ch {
			select {
			case <-ctx.Done():
				return
			case multiplexedStream <- i:
			}
		}
	}

	// Select from all the channels
	wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	// Wait for all the reads to complete
	go func() {
		wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}

// GetCommandsC returns the multiplexed output channel combining all input channels added with AddChannel
func (c *Multiplexer[T]) GetCommandsC() <-chan *T {
	return c.outputCommands
}
