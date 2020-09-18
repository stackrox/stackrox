package networkflow

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

// EntityForDeployment returns a NetworkEntityInfo for the given deployment.
func EntityForDeployment(deployment *storage.ListDeployment) *storage.NetworkEntityInfo {
	return &storage.NetworkEntityInfo{
		Type: storage.NetworkEntityInfo_DEPLOYMENT,
		Id:   deployment.GetId(),
		Desc: &storage.NetworkEntityInfo_Deployment_{
			Deployment: &storage.NetworkEntityInfo_Deployment{
				Name:      deployment.GetName(),
				Namespace: deployment.GetNamespace(),
				Cluster:   deployment.GetCluster(),
			},
		},
	}
}

// PopulateDeploymentDesc populates the entity with deployment information from the given map. It returns false if
// the entity is a deployment with an ID that could not be found in the map, true otherwise (including in case of
// non-deployment entities).
func PopulateDeploymentDesc(entity *storage.NetworkEntityInfo, deploymentsMap map[string]*storage.ListDeployment) bool {
	if entity.GetType() != storage.NetworkEntityInfo_DEPLOYMENT {
		return true
	}
	deployment := deploymentsMap[entity.GetId()]
	if deployment == nil {
		return false
	}
	entity.Desc = &storage.NetworkEntityInfo_Deployment_{
		Deployment: &storage.NetworkEntityInfo_Deployment{
			Name:      deployment.GetName(),
			Namespace: deployment.GetNamespace(),
			Cluster:   deployment.GetCluster(),
		},
	}
	return true
}

// UpdateFlowsWithEntityDesc populates the entity descriptions for source and destination network entities in the
// list of flows. It returns two slices: one containing flows with fully populated information, the other containing
// flows with partially or completely missing deployment entity information.
// Note: Missing external sources are marked as INTERNET.
func UpdateFlowsWithEntityDesc(flows []*storage.NetworkFlow, deployments map[string]*storage.ListDeployment, externalSrcs map[string]*storage.NetworkEntityInfo) (okFlows []*storage.NetworkFlow, missingInfoFlows []*storage.NetworkFlow) {
	okFlows = flows[:0]
	for _, flow := range flows {
		srcOk, dstOk := false, false
		if networkgraph.IsExternal(flow.GetProps().GetSrcEntity()) {
			PopulateExternalSrcsDesc(flow.GetProps().GetSrcEntity(), externalSrcs)
			srcOk = true
		} else {
			srcOk = PopulateDeploymentDesc(flow.GetProps().GetSrcEntity(), deployments)
		}

		if networkgraph.IsExternal(flow.GetProps().GetDstEntity()) {
			PopulateExternalSrcsDesc(flow.GetProps().GetDstEntity(), externalSrcs)
			dstOk = true
		} else {
			dstOk = PopulateDeploymentDesc(flow.GetProps().GetDstEntity(), deployments)
		}

		if srcOk && dstOk {
			okFlows = append(okFlows, flow)
		} else {
			missingInfoFlows = append(missingInfoFlows, flow)
		}
	}

	return
}

// PopulateExternalSrcsDesc populates the entity with external source information from the given map. If external source
// could not be found in the map, it is populated with the de-facto INTERNET entity desc.
// Note: If entity is not EXTERNAL_SOURCE we return true.
func PopulateExternalSrcsDesc(entity *storage.NetworkEntityInfo, externalSrcs map[string]*storage.NetworkEntityInfo) {
	if entity.GetType() != storage.NetworkEntityInfo_EXTERNAL_SOURCE {
		return
	}

	src, ok := externalSrcs[entity.GetId()]
	if ok {
		entity.Desc = src.GetDesc()
		return
	}
	// If the external source (CIDR block) is not visible, mark this entity as INTERNET.
	*entity = *networkgraph.InternetEntity().ToProto()
}
