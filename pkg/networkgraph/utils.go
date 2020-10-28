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
