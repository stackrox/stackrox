package nodescanv2

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

// NodeScanner defines an interface for V2 NodeScanning
type NodeScanner interface {
	Scan(nodeName string) (*storage.NodeScanV2, error)
}

// NodeScan is the V2 NodeScanning implementation
type NodeScan struct {
}

// Scan scans the current node and returns the results as storage.NodeScanV2 object
func (n *NodeScan) Scan(nodeName string) (*storage.NodeScanV2, error) {
	return nil, errors.New("Not implemented")
}
