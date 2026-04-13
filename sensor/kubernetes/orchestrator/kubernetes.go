package orchestrator

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/common/orchestrator"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

type kubernetesOrchestrator struct {
	dynClient dynamic.Interface
}

// New returns a new kubernetes orchestrator client.
func New(dynClient dynamic.Interface) orchestrator.Orchestrator {
	return &kubernetesOrchestrator{
		dynClient: dynClient,
	}
}

func (k *kubernetesOrchestrator) GetNodeScrapeConfig(nodeName string) (*orchestrator.NodeScrapeConfig, error) {
	unstructuredNode, err := k.dynClient.Resource(client.NodeGVR).Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "getting node %q", nodeName)
	}

	var node v1.Node
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredNode.Object, &node); err != nil {
		return nil, errors.Wrap(err, "converting unstructured to node")
	}

	_, hasControlPlaneNodeLabel := node.GetLabels()["node-role.kubernetes.io/control-plane"]

	return &orchestrator.NodeScrapeConfig{
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		IsMasterNode:            hasControlPlaneNodeLabel,
	}, nil
}
