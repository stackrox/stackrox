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

type MessageToComplianceWithAddress struct {
	msg       *sensor.MsgToCompliance
	hostname  string
	broadcast bool
}

type Multiplexer[T any] struct {
	inputChannels  []<-chan *T
	outputCommands chan *T
	//connectionMap  map[string]sensor.ComplianceService_CommunicateServer
	//manager connectionManager

	wg      sync.WaitGroup
	started concurrency.Signal
}

func (c *Multiplexer[T]) Notify(e common.SensorComponentEvent) {
	return
}

func (c *Multiplexer[T]) Start() error {
	// TODO maybe error if this fails(?)
	c.run()
	return nil
}

func (c *Multiplexer[T]) Stop(err error) {
}

func (c *Multiplexer[T]) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *Multiplexer[T]) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (c *Multiplexer[T]) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func NewComplianceCommunicator[T any]() *Multiplexer[T] {
	communicator := Multiplexer[T]{
		inputChannels:  make([]<-chan *T, 0),
		outputCommands: make(chan *T),
		started:        concurrency.Signal{}}

	return &communicator
}

// AddChannel Adds a channel to ComplianceCommunicator, AddChannel must be called
// for ALL channels before calling Run()
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

func (c *Multiplexer[T]) GetCommandsC() <-chan *T {
	return c.outputCommands
}
