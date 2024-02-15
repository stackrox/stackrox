package resources

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	v1 "k8s.io/api/core/v1"
)

type nodeDispatcher struct {
	deploymentStore *DeploymentStore
	nodeStore       nodeStore
	endpointManager endpointManager
}

func newNodeDispatcher(deploymentStore *DeploymentStore, nodeStore nodeStore, endpointManager endpointManager) *nodeDispatcher {
	return &nodeDispatcher{
		deploymentStore: deploymentStore,
		nodeStore:       nodeStore,
		endpointManager: endpointManager,
	}
}

func convertTaints(taints []v1.Taint) []*storage.Taint {
	roxTaints := make([]*storage.Taint, 0, len(taints))
	for _, t := range taints {
		roxTaints = append(roxTaints, &storage.Taint{
			Key:         t.Key,
			Value:       t.Value,
			TaintEffect: k8s.ToRoxTaintEffect(t.Effect),
		})
	}
	return roxTaints
}

func (h *nodeDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) *component.ResourceEvent {
	node := obj.(*v1.Node)
	protoNode := buildNode(node)
	if action == central.ResourceAction_REMOVE_RESOURCE {
		h.nodeStore.removeNode(protoNode)
		h.endpointManager.OnNodeUpdateOrRemove()
	} else {
		wrap := wrapNode(node)

		// Only perform endpoint manager updates if the IP addresses of the node changed.
		if h.nodeStore.addOrUpdateNode(wrap) {
			if action == central.ResourceAction_CREATE_RESOURCE {
				h.endpointManager.OnNodeCreate(wrap)
			} else {
				h.endpointManager.OnNodeUpdateOrRemove()
			}
		}
	}

	return component.NewEvent(&central.SensorEvent{
		Id:     protoNode.GetId(),
		Action: action,
		Resource: &central.SensorEvent_Node{
			Node: protoNode,
		},
	})
}

func buildNode(node *v1.Node) *storage.Node {
	var internal, external []string

	for _, entry := range node.Status.Addresses {
		switch entry.Type {
		case v1.NodeInternalIP:
			internal = append(internal, entry.Address)
		case v1.NodeExternalIP:
			external = append(external, entry.Address)
		}
	}

	creation := node.CreationTimestamp.ProtoTime()
	return &storage.Node{
		Id:                      string(node.UID),
		Name:                    node.Name,
		Taints:                  convertTaints(node.Spec.Taints),
		Labels:                  node.GetLabels(),
		Annotations:             node.GetAnnotations(),
		JoinedAt:                &types.Timestamp{Seconds: creation.Seconds, Nanos: creation.Nanos},
		InternalIpAddresses:     internal,
		ExternalIpAddresses:     external,
		ContainerRuntime:        k8sutil.ParseContainerRuntimeVersion(node.Status.NodeInfo.ContainerRuntimeVersion),
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
		OsImage:                 node.Status.NodeInfo.OSImage,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		KubeProxyVersion:        node.Status.NodeInfo.KubeProxyVersion,
		K8SUpdated:              types.TimestampNow(),
	}
}
