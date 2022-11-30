package nodescanv2

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
	fns, err := (&FakeNodeScanner{}).Scan("someNode")
	n.Nil(err)
	n.NotNil(fns)
	n.IsType(&storage.NodeInventory{}, fns)
}
