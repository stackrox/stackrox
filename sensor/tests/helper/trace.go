package helper

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/sync"
)

// NewNetworkFlowTraceWriter creates a new NetworkFlowTraceWriter.
func NewNetworkFlowTraceWriter(ctx context.Context, messageC chan *sensor.NetworkConnectionInfoMessage) *NetworkFlowTraceWriter {
	return &NetworkFlowTraceWriter{
		ctx:      ctx,
		messageC: messageC,
	}
}

// NetworkFlowTraceWriter writes the network flows received from collector.
type NetworkFlowTraceWriter struct {
	mu       sync.Mutex
	messageC chan *sensor.NetworkConnectionInfoMessage
	ctx      context.Context
}

// Write a slice of bytes in the messageC channel
func (tr *NetworkFlowTraceWriter) Write(data []byte) (int, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	message := &sensor.NetworkConnectionInfoMessage{}
	if err := message.UnmarshalVTUnsafe(data); err != nil {
		return 0, err
	}
	select {
	case <-tr.ctx.Done():
		return 0, errors.New("Context done")
	case tr.messageC <- message:
		return message.SizeVT(), nil
	}
}

// NewProcessIndicatorTraceWriter creates a new ProcessIndicatorTraceWriter.
func NewProcessIndicatorTraceWriter(ctx context.Context, messageC chan *sensor.SignalStreamMessage) *ProcessIndicatorTraceWriter {
	return &ProcessIndicatorTraceWriter{
		ctx:      ctx,
		messageC: messageC,
	}
}

// ProcessIndicatorTraceWriter  writes the network flows received from collector.
type ProcessIndicatorTraceWriter struct {
	mu       sync.Mutex
	messageC chan *sensor.SignalStreamMessage
	ctx      context.Context
}

// Write a slice of bytes in the messageC channel
func (tr *ProcessIndicatorTraceWriter) Write(data []byte) (int, error) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	message := &sensor.SignalStreamMessage{}
	if err := message.UnmarshalVTUnsafe(data); err != nil {
		return 0, err
	}
	select {
	case <-tr.ctx.Done():
		return 0, errors.New("Context done")
	case tr.messageC <- message:
		return message.SizeVT(), nil
	}
}
