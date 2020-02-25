package gatherers

import (
	"context"

	"github.com/stackrox/rox/central/node/globaldatastore"
	"github.com/stackrox/rox/pkg/telemetry/data"
)

type nodeGatherer struct {
	nodeDatastore globaldatastore.GlobalDataStore
}

func newNodeGatherer(nodeDatastore globaldatastore.GlobalDataStore) *nodeGatherer {
	return &nodeGatherer{
		nodeDatastore: nodeDatastore,
	}
}

// Gather returns a list of stats about all the nodes in a cluster
func (n *nodeGatherer) Gather(ctx context.Context, clusterID string) []*data.NodeInfo {
	datastore, err := n.nodeDatastore.GetClusterNodeStore(ctx, clusterID, false)
	if err != nil {
		log.Errorf("unable to get node datastore for cluster %s: %v", clusterID, err)
		return nil
	}
	nodes, err := datastore.ListNodes()
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
