package networkflow

import "github.com/stackrox/rox/generated/storage"

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

// UpdateFlowsWithDeployments populates the entity descriptions for source and destination deployment entities in the
// list of flows. It returns two slices: one containing flows with fully populated information, the other containing
// flows with partially or completely missing deployment information.
func UpdateFlowsWithDeployments(flows []*storage.NetworkFlow, deployments map[string]*storage.ListDeployment) (okFlows []*storage.NetworkFlow, missingInfoFlows []*storage.NetworkFlow) {
	okFlows = flows[:0]
	for _, flow := range flows {
		ok := PopulateDeploymentDesc(flow.GetProps().GetSrcEntity(), deployments)
		if !PopulateDeploymentDesc(flow.GetProps().GetDstEntity(), deployments) {
			ok = false
		}

		if ok {
			okFlows = append(okFlows, flow)
		} else {
			missingInfoFlows = append(missingInfoFlows, flow)
		}
	}

	return
}
