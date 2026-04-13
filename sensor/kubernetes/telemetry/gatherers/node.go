package gatherers

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/telemetry/data"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
)

type nodeGatherer struct {
	dynClient dynamic.Interface
}

func newNodeGatherer(dynClient dynamic.Interface) *nodeGatherer {
	return &nodeGatherer{
		dynClient: dynClient,
	}
}

// Gather returns a list of stats about all the nodes in the cluster this Sensor is monitoring
func (c *nodeGatherer) Gather(ctx context.Context) ([]*data.NodeInfo, error) {
	nodesList, err := c.dynClient.Resource(client.NodeGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "listing Kubernetes nodes")
	}

	nodeInfoList := make([]*data.NodeInfo, 0, len(nodesList.Items))
	for _, item := range nodesList.Items {
		var node v1.Node
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &node); err != nil {
			continue
		}
		adverseConditions := make([]string, 0, len(node.Status.Conditions))
		for _, condition := range node.Status.Conditions {
			if condition.Type == v1.NodeReady && condition.Status == v1.ConditionTrue {
				continue
			}
			if condition.Status == v1.ConditionFalse {
				continue
			}
			adverseConditions = append(adverseConditions, condition.String())
		}
		runtimeVersion := k8sutil.ParseContainerRuntimeVersion(node.Status.NodeInfo.ContainerRuntimeVersion)
		var pType string
		if parts := strings.SplitN(node.Spec.ProviderID, "://", 2); len(parts) == 2 {
			pType = parts[0]
		}

		nodeInfoList = append(nodeInfoList, &data.NodeInfo{
			ID:                      string(node.UID),
			ProviderType:            pType,
			TotalResources:          getResources(node.Status.Capacity),
			AllocatableResources:    getResources(node.Status.Allocatable),
			Unschedulable:           node.Spec.Unschedulable,
			HasTaints:               len(node.Spec.Taints) > 0,
			AdverseConditions:       adverseConditions,
			KernelVersion:           node.Status.NodeInfo.KernelVersion,
			OSImage:                 node.Status.NodeInfo.OSImage,
			ContainerRuntimeVersion: runtimeVersion.GetVersion(),
			KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion:        node.Status.NodeInfo.KubeProxyVersion,
			OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
			Architecture:            node.Status.NodeInfo.Architecture,
			Collector:               nil,
			Compliance:              nil,
		})
	}
	return nodeInfoList, nil
}

func getResources(resources v1.ResourceList) *data.NodeResourceInfo {
	return &data.NodeResourceInfo{
		MilliCores:   int(resources.Cpu().MilliValue()),
		MemoryBytes:  resources.Memory().Value(),
		StorageBytes: resources.StorageEphemeral().Value(),
	}
}
