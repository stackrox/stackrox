package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

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
