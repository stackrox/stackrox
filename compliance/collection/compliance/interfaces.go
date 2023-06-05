package compliance

import (
	"context"

	"github.com/stackrox/rox/compliance/collection/intervals"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// NodeNameProvider provides node name
type NodeNameProvider interface {
	GetNodeName() string
}

// NodeScanner provides a way to obtain a node-inventory
type NodeScanner interface {
	GetIntervals() *intervals.NodeScanIntervals
	ScanNode(ctx context.Context) (*sensor.MsgFromCompliance, error)
	IsActive() bool
}

// UnconfirmedMessageHandler handles the observation of sending, and ACK/NACK messages
type UnconfirmedMessageHandler interface {
	HandleACK()
	HandleNACK()
	ObserveSending()
	RetryCommand() <-chan struct{}
}
