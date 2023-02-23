package nodeinventorizer

import (
	"github.com/stackrox/rox/generated/storage"
)

// NodeInventorizer is the interface that defines the interface a scanner must implement
type NodeInventorizer interface {
	Scan(nodeName string) (*storage.NodeInventory, error)
}
