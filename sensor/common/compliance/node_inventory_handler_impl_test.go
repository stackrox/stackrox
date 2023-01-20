package compliance

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/compliance/mocks"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stretchr/testify/suite"
)

func TestNodeInventoryHandlerImpl(t *testing.T) {
	suite.Run(t, &NodeInventoryHandlerImplTestSuite{})
}

var _ suite.TearDownTestSuite = (*NodeInventoryHandlerTestSuite)(nil)

type NodeInventoryHandlerImplTestSuite struct {
	suite.Suite
	nodeStore store.NodeStore
}

func (s *NodeInventoryHandlerImplTestSuite) SetupTest() {
	s.nodeStore = mocks.NewMockNodeStore()
}

func (s *NodeInventoryHandlerImplTestSuite) TestFindNodeID() {
	dummy := make(chan *storage.NodeInventory)
	defer close(dummy)

	tt := map[string]struct {
		storageState       []*store.NodeWrap
		namesToExpectedIDs map[*storage.NodeInventory]string
	}{
		"Existing node is found": {
			storageState: []*store.NodeWrap{
				{Node: &storage.Node{Id: "id1", Name: "node1"}},
			},
			namesToExpectedIDs: map[*storage.NodeInventory]string{
				{NodeName: "node1"}: "id1",
				{NodeName: "node2"}: "",
			},
		},
		"Empty store": {
			storageState: []*store.NodeWrap{},
			namesToExpectedIDs: map[*storage.NodeInventory]string{
				{NodeName: "node1"}: "",
				{NodeName: "node2"}: "",
			},
		},
		"Node got replaced and kept the name but changed ID": {
			storageState: []*store.NodeWrap{
				{Node: &storage.Node{Id: "id1", Name: "node1"}},
				{Node: &storage.Node{Id: "id2", Name: "node2"}},
				{Node: &storage.Node{Id: "id7", Name: "node1"}}, // node gets replaced
			},
			namesToExpectedIDs: map[*storage.NodeInventory]string{
				{NodeName: "node1"}: "id7",
				{NodeName: "node2"}: "id2",
			},
		},
		"Node changend name but kept ID": {
			storageState: []*store.NodeWrap{
				{Node: &storage.Node{Id: "id1", Name: "node1"}},
				{Node: &storage.Node{Id: "id1", Name: "node7"}}, // node got renamed
			},
			namesToExpectedIDs: map[*storage.NodeInventory]string{
				{NodeName: "node1"}: "id1",
				{NodeName: "node7"}: "id1",
			},
		},
	}
	for name, tc := range tt {
		s.Run(name, func() {
			// cleanup store state
			s.nodeStore = mocks.NewMockNodeStore()
			// populate store with data
			for _, nwrap := range tc.storageState {
				s.nodeStore.AddOrUpdateNode(nwrap)
			}

			// create handler with mocked nodeStore
			h := NewNodeInventoryHandler(dummy, NewNodeIDMatcher(s.nodeStore))

			for inventory, expectedID := range tc.namesToExpectedIDs {
				gotID := h.findNodeID(inventory)
				s.Equal(expectedID, gotID, "ID mismatch for inventory '%s'", inventory.GetNodeName())
			}
		})
	}

}
