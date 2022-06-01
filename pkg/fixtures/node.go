package fixtures

import (
	"fmt"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// GetNode returns a mock *storage.Node
func GetNode() *storage.Node {
	componentCount := 50
	components := make([]*storage.EmbeddedNodeScanComponent, 0, componentCount)
	for i := 0; i < componentCount; i++ {
		components = append(components, &storage.EmbeddedNodeScanComponent{
			Name:    "name",
			Version: "1.2.3.4",
			Vulns:   getVulnsPerComponent(i, storage.EmbeddedVulnerability_NODE_VULNERABILITY),
		})
	}
	return getNodeWithComponents(components)
}

// GetNodeWithUniqueComponents returns a mock Node where each component is unique
func GetNodeWithUniqueComponents() *storage.Node {
	componentCount := 2
	components := make([]*storage.EmbeddedNodeScanComponent, 0, componentCount)
	for i := 0; i < componentCount; i++ {
		components = append(components, &storage.EmbeddedNodeScanComponent{
			Name:    fmt.Sprintf("name-%d", i),
			Version: fmt.Sprintf("%d.2.3.4", i),
			Vulns:   getVulnsPerComponent(i, storage.EmbeddedVulnerability_NODE_VULNERABILITY),
		})
	}
	return getNodeWithComponents(components)
}

func getNodeWithComponents(components []*storage.EmbeddedNodeScanComponent) *storage.Node {
	return &storage.Node{
		Id:   "id",
		Name: "name",
		Scan: &storage.NodeScan{
			ScanTime:   types.TimestampNow(),
			Components: components,
		},
	}
}

func GetScopedNode(ID string, clusterID string) *storage.Node {
	return &storage.Node{
		Id:          ID,
		Name:        ID,
		ClusterId:   clusterID,
		ClusterName: clusterID,
		ContainerRuntime: &storage.ContainerRuntimeInfo{
			Type:    storage.ContainerRuntime_DOCKER_CONTAINER_RUNTIME,
			Version: "20.10.10",
		},
		JoinedAt:        &types.Timestamp{Seconds: 1654103379},
		OperatingSystem: "Docker Desktop",
		Scan:            generateNodeScan(),
	}
}

func generateNodeScan() *storage.NodeScan {
	return &storage.NodeScan{
		ScanTime:        &types.Timestamp{Seconds: 1654103579},
		OperatingSystem: "Linux",
		Components:      generateNodeScanComponents(),
	}
}

func generateNodeScanComponents() []*storage.EmbeddedNodeScanComponent {
	return nil
}

