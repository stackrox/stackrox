package node

import (
	"context"

	"github.com/stackrox/rox/compliance/utils"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// NodeNameProvider provides node name
type NodeNameProvider interface {
	GetNodeName() string
}

// NodeScanner provides a way to obtain a node-inventory
type NodeScanner interface {
	GetIntervals() *utils.NodeScanIntervals
	ScanNode(ctx context.Context) (*sensor.MsgFromCompliance, error)
	IsActive() bool
}

// NodeIndexer represents a node indexer.
//
// It is a specialized mode of Scanners Indexer that takes a path and scans a live filesystem
// instead of downloading and scanning layers of a container manifest.
type NodeIndexer interface {
	IndexNode(ctx context.Context) (*v4.IndexReport, error)
	GetIntervals() *utils.NodeScanIntervals
}

// UnconfirmedMessageHandler handles the observation of sending, and ACK/NACK messages.
// Each resource (identified by resourceID) has independent retry state.
type UnconfirmedMessageHandler interface {
	HandleACK(resourceID string)
	HandleNACK(resourceID string)
	ObserveSending(resourceID string)
	RetryCommand() <-chan string // Returns resourceID to retry
	OnACK(callback func(resourceID string))
}
