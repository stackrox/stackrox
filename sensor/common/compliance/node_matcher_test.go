package compliance

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

func TestNodeIDMatcher(t *testing.T) {
	suite.Run(t, &NodeInventoryHandlerImplTestSuite{})
}

var _ suite.TearDownTestSuite = (*NodeInventoryHandlerTestSuite)(nil)

type NodeInventoryHandlerImplTestSuite struct {
	suite.Suite
}

func (s *NodeInventoryHandlerImplTestSuite) TestNodeIDMatcherGetNodeID() {
	dummy := make(chan *storage.NodeInventory)
	defer close(dummy)

	tt := map[string]struct {
		storageState       map[string]string
		namesToExpectedIDs map[string]string
	}{
		"Existing node is found": {
			storageState: map[string]string{
				"node1": "id1",
			},
			namesToExpectedIDs: map[string]string{
				"node1": "id1",
				"node2": "",
			},
		},
		"Empty store": {
			storageState: map[string]string{},
			namesToExpectedIDs: map[string]string{
				"node1": "",
				"node2": "",
			},
		},
	}
	for name, tc := range tt {
		s.Run(name, func() {
			matcher := newMockNodeIDMatcher(tc.storageState)
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

// mockNodeIDMatcher always finds a node when GetNodeResource is called
type mockNodeIDMatcher struct {
	nodeStore map[string]string
}

// newMockNodeIDMatcher builds mockNodeIDMatcher
func newMockNodeIDMatcher(store map[string]string) *mockNodeIDMatcher {
	return &mockNodeIDMatcher{
		nodeStore: store,
	}
}

// GetNodeID searches for nodeID in the map and returns it when found
func (c *mockNodeIDMatcher) GetNodeID(nodename string) (string, error) {
	if val, ok := c.nodeStore[nodename]; ok {
		return val, nil
	}
	return "", errors.New("node not found")
}
