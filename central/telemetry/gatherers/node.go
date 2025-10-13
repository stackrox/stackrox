package gatherers

import (
	"context"

	"github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type nodeGatherer struct {
	nodeDatastore datastore.DataStore
}

func newNodeGatherer(nodeDatastore datastore.DataStore) *nodeGatherer {
	return &nodeGatherer{
		nodeDatastore: nodeDatastore,
	}
}

// Gather returns a list of stats about all the nodes in a cluster
func (n *nodeGatherer) Gather(ctx context.Context, clusterID string) []*data.NodeInfo {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	nodes, err := n.nodeDatastore.SearchRawNodes(ctx, q)
	if err != nil {
		log.Errorf("unable to get nodes for cluster %s: %v", clusterID, err)
		return nil
	}
	nodeList := make([]*data.NodeInfo, 0, len(nodes))
	for _, node := range nodes {
		runtimeVersion := node.GetContainerRuntime().GetVersion()
		// This is the subset of things the Sensor reports to Central.  We fall back to this if we can't reach the
		// Sensor
		nodeList = append(nodeList, &data.NodeInfo{
			ID:                      node.GetId(),
			HasTaints:               len(node.GetTaints()) > 0,
			KernelVersion:           node.GetKernelVersion(),
			OSImage:                 node.GetOsImage(),
			ContainerRuntimeVersion: runtimeVersion,
			KubeletVersion:          node.GetKubeletVersion(),
		})
	}
	return nodeList
}
