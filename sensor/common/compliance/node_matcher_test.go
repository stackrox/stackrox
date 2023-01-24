package compliance

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/compliance/mocks"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stretchr/testify/suite"
)

func TestNodeIDMatcher(t *testing.T) {
	suite.Run(t, &NodeInventoryHandlerImplTestSuite{})
}

var _ suite.TearDownTestSuite = (*NodeInventoryHandlerTestSuite)(nil)

type NodeInventoryHandlerImplTestSuite struct {
	suite.Suite
	nodeStore nodeStore
}

func (s *NodeInventoryHandlerImplTestSuite) SetupTest() {
	s.nodeStore = mocks.NewMockNodeStore()
}

func (s *NodeInventoryHandlerImplTestSuite) TestNodeIDMatcherGetNodeID() {
	dummy := make(chan *storage.NodeInventory)
	defer close(dummy)

	tt := map[string]struct {
		storageState       []*store.NodeWrap
		namesToExpectedIDs map[string]string
	}{
		"Existing node is found": {
			storageState: []*store.NodeWrap{
				{Node: &storage.Node{Id: "id1", Name: "node1"}},
			},
			namesToExpectedIDs: map[string]string{
				"node1": "id1",
				"node2": "",
			},
		},
		"Empty store": {
			storageState: []*store.NodeWrap{},
			namesToExpectedIDs: map[string]string{
				"node1": "",
				"node2": "",
			},
		},
		"Node got replaced and kept the name but changed ID": {
			storageState: []*store.NodeWrap{
				{Node: &storage.Node{Id: "id1", Name: "node1"}},
				{Node: &storage.Node{Id: "id2", Name: "node2"}},
				{Node: &storage.Node{Id: "id7", Name: "node1"}}, // node gets replaced
			},
			namesToExpectedIDs: map[string]string{
				"node1": "id7",
				"node2": "id2",
			},
		},
		"Node changend name but kept ID": {
			storageState: []*store.NodeWrap{
				{Node: &storage.Node{Id: "id1", Name: "node1"}},
				{Node: &storage.Node{Id: "id1", Name: "node7"}}, // node got renamed
			},
			namesToExpectedIDs: map[string]string{
				"node1": "id1",
				"node7": "id1",
			},
		},
	}
	for name, tc := range tt {
		s.Run(name, func() {
			// cleanup store state
			s.nodeStore = mocks.NewMockNodeStore()
			matcher := NewNodeIDMatcher(s.nodeStore)
			// populate store with data
			for _, nwrap := range tc.storageState {
				s.nodeStore.AddOrUpdateNode(nwrap)
			}

			for nodeName, expectedID := range tc.namesToExpectedIDs {
				gotID, gotErr := matcher.GetNodeID(nodeName)
				s.Equal(expectedID, gotID, "ID mismatch for inventory '%s'", nodeName)
				if gotID != "" {
					s.Nil(gotErr)
				} else {
					s.NotNil(gotErr)
				}
			}
		})
	}

}
