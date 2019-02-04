package resources

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv/k8s"
	"k8s.io/api/core/v1"
)

type nodeDispatcher struct {
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	nodeStore       *nodeStore
	endpointManager *endpointManager
}

func newNodeDispatcher(serviceStore *serviceStore, deploymentStore *deploymentStore, nodeStore *nodeStore, endpointManager *endpointManager) *nodeDispatcher {
	return &nodeDispatcher{
		serviceStore:    serviceStore,
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

func (h *nodeDispatcher) ProcessEvent(obj interface{}, action central.ResourceAction) []*central.SensorEvent {
	node := obj.(*v1.Node)
	if action == central.ResourceAction_REMOVE_RESOURCE {
		h.nodeStore.removeNode(node)
	} else {
		wrap := wrapNode(node)
		h.nodeStore.addOrUpdateNode(wrap)

		if action == central.ResourceAction_CREATE_RESOURCE {
			h.endpointManager.OnNodeCreate(wrap)
		}
	}

	if action != central.ResourceAction_CREATE_RESOURCE {
		h.endpointManager.OnNodeUpdateOrRemove(node.Name)
	}

	var internal, external []string

	for _, entry := range node.Status.Addresses {
		switch entry.Type {
		case v1.NodeInternalIP:
			internal = append(internal, entry.Address)
		case v1.NodeExternalIP:
			external = append(external, entry.Address)
		}
	}

	creation := (&node.CreationTimestamp).ProtoTime()
	nodeResource := &storage.Node{
		Id:                      string(node.UID),
		Name:                    node.Name,
		Taints:                  convertTaints(node.Spec.Taints),
		Labels:                  node.GetLabels(),
		Annotations:             node.GetAnnotations(),
		JoinedAt:                &types.Timestamp{Seconds: creation.Seconds, Nanos: creation.Nanos},
		InternalIpAddresses:     internal,
		ExternalIpAddresses:     external,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		OsImage:                 node.Status.NodeInfo.OSImage,
	}

	events := []*central.SensorEvent{
		{
			Id:     nodeResource.GetId(),
			Action: action,
			Resource: &central.SensorEvent_Node{
				Node: nodeResource,
			},
		},
	}

	return events
}
