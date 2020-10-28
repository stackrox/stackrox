package networkgraph

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
)

// IsDeployment returns true if the network entity is a deployment (by type).
func IsDeployment(entity *storage.NetworkEntityInfo) bool {
	return entity.GetType() == storage.NetworkEntityInfo_DEPLOYMENT
}

// IsExternal returns true if the network entity is external to cluster (by type).
func IsExternal(entity *storage.NetworkEntityInfo) bool {
	return IsKnownExternalSrc(entity) || entity.GetType() == storage.NetworkEntityInfo_INTERNET
}

// IsKnownDefaultExternal returns true if the network entity is known system-generated network source.
// Note: INTERNET is not treated as system-generated but rather a fallback when exact data is unavailable.
func IsKnownDefaultExternal(entity *storage.NetworkEntityInfo) bool {
	return IsKnownExternalSrc(entity) && entity.GetExternalSource().GetDefault()
}

// IsKnownExternalSrc returns true if the network entity is known external source.
func IsKnownExternalSrc(entity *storage.NetworkEntityInfo) bool {
	return entity.GetType() == storage.NetworkEntityInfo_EXTERNAL_SOURCE
}

// AnyExternal returns true if at least one network entity is external to cluster (by type).
func AnyExternal(src, dst *storage.NetworkEntityInfo) bool {
	return IsExternal(src) || IsExternal(dst)
}

// AllExternal returns true iff both network entities are external to cluster (by type).
func AllExternal(src, dst *storage.NetworkEntityInfo) bool {
	return IsExternal(src) && IsExternal(dst)
}

// AnyExternalInFilter accepts two network entities, source and destination, and external network entity ID set, and returns true if
// input set contains at least one endpoint and is external to cluster. Note: We regard UNKNOWN and LISTEN_ENDPOINTS as invisible.
func AnyExternalInFilter(src, dst *storage.NetworkEntityInfo, filter set.StringSet) bool {
	if IsExternal(src) && filter.Contains(src.GetId()) {
		return true
	}
	if IsExternal(dst) && filter.Contains(dst.GetId()) {
		return true
	}
	return false
}

// AnyDeployment returns true if at least one network entity is a deployment (by type).
func AnyDeployment(src, dst *storage.NetworkEntityInfo) bool {
	return IsDeployment(src) || IsDeployment(dst)
}

// AnyDeploymentInFilter accepts two network entities, source and destination, and deployments map, and returns true if
// input map contains at least one endpoint and is a deployment. Note: We regard UNKNOWN and LISTEN_ENDPOINTS as invisible.
func AnyDeploymentInFilter(src, dst *storage.NetworkEntityInfo, filter map[string]*storage.ListDeployment) bool {
	if IsDeployment(src) && filter[src.GetId()] != nil {
		return true
	}
	if IsDeployment(dst) && filter[dst.GetId()] != nil {
		return true
	}
	return false
}

// NetworkEntityForDeployment returns a NetworkEntityInfo for the given deployment.
func NetworkEntityForDeployment(deployment *storage.ListDeployment) *storage.NetworkEntityInfo {
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
		if IsExternal(flow.GetProps().GetSrcEntity()) {
			PopulateExternalSrcsDesc(flow.GetProps().GetSrcEntity(), externalSrcs)
			srcOk = true
		} else {
			srcOk = PopulateDeploymentDesc(flow.GetProps().GetSrcEntity(), deployments)
		}

		if IsExternal(flow.GetProps().GetDstEntity()) {
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
	*entity = *InternetEntity().ToProto()
}
