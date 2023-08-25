package fixtures

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

// GetNode returns a mock *storage.Node
func GetNode() *storage.Node {
	componentCount := 50
	components := make([]*storage.EmbeddedNodeScanComponent, 0, componentCount)
	for i := 0; i < componentCount; i++ {
		components = append(components, &storage.EmbeddedNodeScanComponent{
			Name:    "name",
			Version: "1.2.3.4",
			Vulns:   getVulnsPerComponent(i, 5, storage.EmbeddedVulnerability_NODE_VULNERABILITY),
		})
	}
	return getNodeWithComponents(components)
}

// GetNodeWithUniqueComponents returns a mock Node where each component is unique
func GetNodeWithUniqueComponents(numComponents, numVulns int) *storage.Node {
	components := make([]*storage.EmbeddedNodeScanComponent, 0, numComponents)
	for i := 0; i < numComponents; i++ {
		components = append(components, &storage.EmbeddedNodeScanComponent{
			Name:    fmt.Sprintf("name-%d", i),
			Version: fmt.Sprintf("%d.2.3.4", i),
			Vulns:   getVulnsPerComponent(i, numVulns, storage.EmbeddedVulnerability_NODE_VULNERABILITY),
		})
	}
	return getNodeWithComponents(components)
}

func getNodeWithComponents(components []*storage.EmbeddedNodeScanComponent) *storage.Node {
	return &storage.Node{
		Id:   fixtureconsts.Node1,
		Name: "name",
		Scan: &storage.NodeScan{
			ScanTime:   types.TimestampNow(),
			Components: components,
		},
		SetComponents: &storage.Node_Components{
			Components: int32(len(components)),
		},
	}
}

// GetScopedNode returns a mock Node belonging to the input scope.
func GetScopedNode(ID string, clusterID string) *storage.Node {
	return &storage.Node{
		Id:        ID,
		ClusterId: clusterID,
	}
}
