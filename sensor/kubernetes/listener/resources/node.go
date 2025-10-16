package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"google.golang.org/protobuf/proto"
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
		taint := &storage.Taint{}
		taint.SetKey(t.Key)
		taint.SetValue(t.Value)
		taint.SetTaintEffect(k8s.ToRoxTaintEffect(t.Effect))
		roxTaints = append(roxTaints, taint)
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

	se := &central.SensorEvent{}
	se.SetId(protoNode.GetId())
	se.SetAction(action)
	se.SetNode(proto.ValueOrDefault(protoNode))
	return component.NewEvent(se)
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
	node2 := &storage.Node{}
	node2.SetId(string(node.UID))
	node2.SetName(node.Name)
	node2.SetTaints(convertTaints(node.Spec.Taints))
	node2.SetLabels(node.GetLabels())
	node2.SetAnnotations(node.GetAnnotations())
	node2.SetJoinedAt(protocompat.GetProtoTimestampFromSecondsAndNanos(creation.Seconds, creation.Nanos))
	node2.SetInternalIpAddresses(internal)
	node2.SetExternalIpAddresses(external)
	node2.SetContainerRuntime(k8sutil.ParseContainerRuntimeVersion(node.Status.NodeInfo.ContainerRuntimeVersion))
	node2.SetContainerRuntimeVersion(node.Status.NodeInfo.ContainerRuntimeVersion)
	node2.SetKernelVersion(node.Status.NodeInfo.KernelVersion)
	node2.SetOperatingSystem(node.Status.NodeInfo.OperatingSystem)
	node2.SetOsImage(node.Status.NodeInfo.OSImage)
	node2.SetKubeletVersion(node.Status.NodeInfo.KubeletVersion)
	node2.SetKubeProxyVersion(node.Status.NodeInfo.KubeProxyVersion)
	node2.SetK8SUpdated(protocompat.TimestampNow())
	return node2
}
