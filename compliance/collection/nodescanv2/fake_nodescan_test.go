package nodescanv2

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestFakeNodeScan(t *testing.T) {
	suite.Run(t, &NodeScanSuite{})
}

type FakeNodeScanSuite struct {
	suite.Suite
}

func (n *FakeNodeScanSuite) TestMessageFormat() {
	fns, err := (&FakeNodeScanner{}).Scan("someNode")
	n.Nil(err)
	n.NotNil(fns)
	n.IsType(&storage.NodeInventory{}, fns)
}
