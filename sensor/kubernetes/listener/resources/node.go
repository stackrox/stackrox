package resources

import (
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"k8s.io/api/core/v1"
)

type nodeHandler struct {
	serviceStore    *serviceStore
	deploymentStore *deploymentStore
	nodeStore       *nodeStore
	endpointManager *endpointManager
}

func newNodeHandler(serviceStore *serviceStore, deploymentStore *deploymentStore, nodeStore *nodeStore, endpointManager *endpointManager) *nodeHandler {
	return &nodeHandler{
		serviceStore:    serviceStore,
		deploymentStore: deploymentStore,
		nodeStore:       nodeStore,
		endpointManager: endpointManager,
	}
}

func (h *nodeHandler) Process(node *v1.Node, action pkgV1.ResourceAction) []*pkgV1.SensorEvent {
	if action == pkgV1.ResourceAction_REMOVE_RESOURCE {
		h.nodeStore.removeNode(node)
	} else {
		wrap := wrapNode(node)
		h.nodeStore.addOrUpdateNode(wrap)

		if action == pkgV1.ResourceAction_CREATE_RESOURCE {
			h.endpointManager.OnNodeCreate(wrap)
		}
	}

	if action != pkgV1.ResourceAction_CREATE_RESOURCE {
		h.endpointManager.OnNodeUpdateOrRemove(node.Name)
	}

	nodeResource := &storage.Node{
		Id:   string(node.UID),
		Name: node.Name,
	}

	events := []*pkgV1.SensorEvent{
		{
			Id:     nodeResource.GetId(),
			Action: action,
			Resource: &pkgV1.SensorEvent_Node{
				Node: nodeResource,
			},
		},
	}

	return events
}
