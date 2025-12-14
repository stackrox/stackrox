package main

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	firstValidCID  = 3     // First valid vsock CID (0=hypervisor, 1=loopback, 2=host)
	vmsPerNodeSlot = 10000 // Max VMs per node partition (for spacing)
)

// cidRange contains the calculated CID range for this node.
type cidRange struct {
	StartCID    uint32
	EndCID      uint32
	NodeIndex   int
	TotalNodes  int
	VMsThisNode int
}

// calculateCIDRange determines the CID range for this pod based on its node's index
// in the sorted list of worker nodes. This ensures no overlap between DaemonSet pods.
// vmCountTotal is the total number of VMs across ALL nodes in the cluster.
func calculateCIDRange(ctx context.Context, nodeName string, vmCountTotal int) (cidRange, error) {
	clientset, err := createK8sClient()
	if err != nil {
		return cidRange{}, err
	}

	nodeNames, err := listWorkerNodes(ctx, clientset)
	if err != nil {
		return cidRange{}, err
	}

	return computeCIDRange(nodeNames, nodeName, vmCountTotal)
}

func createK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("get cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return clientset, nil
}

func listWorkerNodes(ctx context.Context, clientset *kubernetes.Clientset) ([]string, error) {
	// Try worker label first
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	if len(nodes.Items) > 0 {
		names := make([]string, 0, len(nodes.Items))
		for _, node := range nodes.Items {
			names = append(names, node.Name)
		}
		return names, nil
	}

	// Fall back to all nodes excluding control plane (for GKE, etc.)
	log.Info("no nodes with worker label found, listing all nodes and excluding control plane")
	nodes, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list all nodes: %w", err)
	}

	var workerNodes []string
	for _, node := range nodes.Items {
		labels := node.GetLabels()
		if _, hasControlPlane := labels["node-role.kubernetes.io/control-plane"]; hasControlPlane {
			continue
		}
		if _, hasMaster := labels["node-role.kubernetes.io/master"]; hasMaster {
			continue
		}
		workerNodes = append(workerNodes, node.Name)
	}

	if len(workerNodes) == 0 {
		return nil, fmt.Errorf("no worker nodes found (found %d total nodes, all are control plane)", len(nodes.Items))
	}

	log.Infof("found %d worker nodes out of %d total nodes", len(workerNodes), len(nodes.Items))
	return workerNodes, nil
}

func computeCIDRange(nodeNames []string, nodeName string, vmCountTotal int) (cidRange, error) {
	if len(nodeNames) == 0 {
		return cidRange{}, fmt.Errorf("no worker nodes found")
	}

	sort.Strings(nodeNames)
	totalNodes := len(nodeNames)

	nodeIndex := -1
	for i, name := range nodeNames {
		if name == nodeName {
			nodeIndex = i
			break
		}
	}

	if nodeIndex == -1 {
		return cidRange{}, fmt.Errorf("node %s not found in worker node list", nodeName)
	}

	vmsPerNode := vmCountTotal / totalNodes
	remainder := vmCountTotal % totalNodes

	vmsThisNode := vmsPerNode
	if nodeIndex < remainder {
		vmsThisNode++
	}

	if vmsThisNode > vmsPerNodeSlot {
		return cidRange{}, fmt.Errorf("too many VMs per node: %d VMs/node exceeds capacity of %d (reduce vmCount or add more nodes)", vmsThisNode, vmsPerNodeSlot)
	}

	startCID := uint32(firstValidCID) + uint32(nodeIndex*vmsPerNodeSlot)
	endCID := startCID + uint32(vmsThisNode) - 1

	const maxCID = uint32(4294967295)
	if endCID > maxCID {
		return cidRange{}, fmt.Errorf("CID overflow: endCID %d exceeds maximum %d (too many nodes or VMs)", endCID, maxCID)
	}

	return cidRange{
		StartCID:    startCID,
		EndCID:      endCID,
		NodeIndex:   nodeIndex,
		TotalNodes:  totalNodes,
		VMsThisNode: vmsThisNode,
	}, nil
}
