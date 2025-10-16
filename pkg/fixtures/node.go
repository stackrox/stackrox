package fixtures

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
)

// GetNode returns a mock *storage.Node
func GetNode() *storage.Node {
	componentCount := 50
	components := make([]*storage.EmbeddedNodeScanComponent, 0, componentCount)
	for i := 0; i < componentCount; i++ {
		ensc := &storage.EmbeddedNodeScanComponent{}
		ensc.SetName("name")
		ensc.SetVersion("1.2.3.4")
		ensc.SetVulns(getVulnsPerComponent(i, 5, storage.EmbeddedVulnerability_NODE_VULNERABILITY))
		components = append(components, ensc)
	}
	return getNodeWithComponents(components)
}

// GetNodeWithUniqueComponents returns a mock Node where each component is unique
func GetNodeWithUniqueComponents(numComponents, numVulns int) *storage.Node {
	components := make([]*storage.EmbeddedNodeScanComponent, 0, numComponents)
	for i := 0; i < numComponents; i++ {
		ensc := &storage.EmbeddedNodeScanComponent{}
		ensc.SetName(fmt.Sprintf("name-%d", i))
		ensc.SetVersion(fmt.Sprintf("%d.2.3.4", i))
		ensc.SetVulns(getVulnsPerComponent(i, numVulns, storage.EmbeddedVulnerability_NODE_VULNERABILITY))
		components = append(components, ensc)
	}
	return getNodeWithComponents(components)
}

func getNodeWithComponents(components []*storage.EmbeddedNodeScanComponent) *storage.Node {
	nodeScan := &storage.NodeScan{}
	nodeScan.SetScanTime(protocompat.TimestampNow())
	nodeScan.SetComponents(components)
	node := &storage.Node{}
	node.SetId(fixtureconsts.Node1)
	node.SetName("name")
	node.SetScan(nodeScan)
	node.Set_Components(int32(len(components)))
	return node
}

// GetScopedNode returns a mock Node belonging to the input scope.
func GetScopedNode(ID string, clusterID string) *storage.Node {
	node := &storage.Node{}
	node.SetId(ID)
	node.SetClusterId(clusterID)
	return node
}
