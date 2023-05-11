package compliance

import (
	"context"

	"github.com/stackrox/rox/compliance/collection/intervals"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

type NodeNameProvider interface {
	GetNodeName() string
}

type NodeScanner interface {
	ManageNodeScanLoop(ctx context.Context, i intervals.NodeScanIntervals) <-chan *sensor.MsgFromCompliance
	ScanNode(ctx context.Context) (*sensor.MsgFromCompliance, error)
	IsActive() bool
}

type SensorReplyHandler interface {
	HandleACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient)
	HandleNACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient)
}
