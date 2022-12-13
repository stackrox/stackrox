package nodeinventorizer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestNodeScan(t *testing.T) {
	suite.Run(t, &NodeScanSuite{})
}

type NodeScanSuite struct {
	suite.Suite
}

func (n *NodeScanSuite) TestMessageFormat() {
	inventory, err := (&NodeInventoryCollector{}).Scan("someNode")
	n.Nil(err)
	n.NotNil(inventory)
	n.IsType(&storage.NodeInventory{}, inventory)
}

// TODO: Test conversion functions and real Scan!
