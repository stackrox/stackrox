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

type Multiplexer struct {
	inputChannels  []<-chan *MessageToComplianceWithAddress
	outputCommands chan *MessageToComplianceWithAddress
	//connectionMap  map[string]sensor.ComplianceService_CommunicateServer
	//manager connectionManager

	wg      sync.WaitGroup
	started concurrency.Signal
}

func (c *Multiplexer) Notify(e common.SensorComponentEvent) {
	return
}

func (c *Multiplexer) Start() error {
	// TODO maybe error if this fails(?)
	c.run()
	return nil
}

func (c *Multiplexer) Stop(err error) {
}

func (c *Multiplexer) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (c *Multiplexer) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (c *Multiplexer) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func NewComplianceCommunicator() *Multiplexer {
	communicator := Multiplexer{
		inputChannels:  make([]<-chan *MessageToComplianceWithAddress, 0),
		outputCommands: make(chan *MessageToComplianceWithAddress),
		wg:             sync.WaitGroup{},
		started:        concurrency.Signal{}}

	return &communicator
}

// AddChannel Adds a channel to ComplianceCommunicator, AddChannel must be called
// for ALL channels before calling Run()
func (c *Multiplexer) AddChannel(channel <-chan *MessageToComplianceWithAddress) {
	if c.started.IsDone() {
		panic("Cannot AddChannel after component is started")
	}
	c.inputChannels = append(c.inputChannels, channel)
}

func (c *Multiplexer) run() {
	c.started.Signal()
	ctx := context.Background()

	output := c.fanIn(ctx, c.inputChannels...)
	for o := range output {
		c.outputCommands <- o
	}
}

func (c *Multiplexer) fanIn(ctx context.Context, channels ...<-chan *MessageToComplianceWithAddress) <-chan *MessageToComplianceWithAddress {
	multiplexedStream := make(chan *MessageToComplianceWithAddress)

	multiplex := func(ch <-chan *MessageToComplianceWithAddress) {
		defer c.wg.Done()
		for i := range ch {
			select {
			case <-ctx.Done():
				return
			case multiplexedStream <- i:
			}
		}
	}

	// Select from all the channels
	c.wg.Add(len(channels))
	for _, c := range channels {
		go multiplex(c)
	}

	// Wait for all the reads to complete
	go func() {
		c.wg.Wait()
		close(multiplexedStream)
	}()

	return multiplexedStream
}

func (c *Multiplexer) GetCommandsC() <-chan *MessageToComplianceWithAddress {
	return c.outputCommands
}
