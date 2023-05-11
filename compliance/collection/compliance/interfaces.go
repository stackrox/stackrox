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
	ManageNodeScanLoop(ctx context.Context, i intervals.NodeScanIntervals) <-chan *sensor.MsgFromCompliance
	ScanNode(ctx context.Context) (*sensor.MsgFromCompliance, error)
	IsActive() bool
}

// SensorReplyHandler handles the ack/nack message from Sensor
type SensorReplyHandler interface {
	HandleACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient)
	HandleNACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient)
}
