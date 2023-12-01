package channelmultiplexer

import (
	"context"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

// ChannelMultiplexer combines n input channels of type T into one output channel of type T
type ChannelMultiplexer[T any] struct {
	inputChannels  []<-chan T
	outputCommands chan T

	started concurrency.Signal
}

// NewMultiplexer creates a ChannelMultiplexer of type T
func NewMultiplexer[T any]() *ChannelMultiplexer[T] {
	multiplexer := ChannelMultiplexer[T]{
		inputChannels:  make([]<-chan T, 0),
		outputCommands: make(chan T),
		started:        concurrency.NewSignal()}

	return &multiplexer
}

// AddChannel Adds a channel to ComplianceCommunicator, AddChannel must be called
// for ALL channels before calling Run()
func (c *ChannelMultiplexer[T]) AddChannel(channel <-chan T) {
	if c.started.IsDone() {
		panic("channelMultiplexer.AddChannel() was called after the component has started. Channels should be added before starting the component")
	}
	c.inputChannels = append(c.inputChannels, channel)
}

// Run starts the ChannelMultiplexer. Make sure to only call Run after all AddChannel calls
func (c *ChannelMultiplexer[T]) Run() {
	c.started.Signal()
	ctx := context.Background()

	output := FanIn[T](ctx, c.inputChannels...)
	go func() {
		for o := range output {
			c.outputCommands <- o
		}
	}()
}

// GetOutput returns the multiplexed output channel combining all input channels added with AddChannel
func (c *ChannelMultiplexer[T]) GetOutput() <-chan T {
	return c.outputCommands
}

// FanIn multiplexes multiple input channels into one output channel and
// finishes when all input channels are closed
func FanIn[T any](ctx context.Context, channels ...<-chan T) <-chan T {
	multiplexedStream := make(chan T)
	wg := sync.WaitGroup{}

	multiplex := func(ch <-chan T) {
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
