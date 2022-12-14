package centralclient

import (
	"strings"
	"sync"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	trackedPaths set.FrozenSet[string]
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/"}

	uninitializedClusters = set.NewSet[string]()
	mux                   = &sync.Mutex{}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call":            {apiCall},
		"Cluster Registered":  {clusterRegistered},
		"Cluster Initialized": {clusterInitialized},
		"roxctl":              {roxctl},
	}
)

// Adds Path, Code and User-Agent properties to API Call events for the API
// paths which start from the prefixes specified in the
// rhacs.redhat.com/telemetry-apipaths central deployment annotation
// ("*" value enables all paths) and are not in the ignoredPath list.
func apiCall(rp *phonehome.RequestParams, props map[string]any) bool {
	for _, ip := range ignoredPaths {
		if strings.HasPrefix(rp.Path, ip) {
			return false
		}
	}
	if trackedPaths.Contains("*") || trackedPaths.Contains(rp.Path) {
		props["Path"] = rp.Path
		props["Code"] = rp.Code
		props["User-Agent"] = rp.UserAgent
		props["Method"] = rp.GetMethod()
		props["Protocol"] = rp.GetProtocol()
		return true
	}
	return false
}

// Adds specific properties to the Cluster Registered event.
func clusterRegistered(rp *phonehome.RequestParams, props map[string]any) bool {
	if rp.Path != "/v1.ClustersService/PostCluster" {
		return false
	}
	props["Code"] = rp.Code
	if req, ok := rp.GRPCReq.(*storage.Cluster); ok {
		props["Cluster Type"] = req.GetType().String()
		props["Cluster ID"] = req.GetId()
		props["Managed By"] = req.GetManagedBy().String()
		mux.Lock()
		defer mux.Unlock()
		if req.GetHealthStatus().GetSensorHealthStatus() == storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Add(req.GetId())
		}
	}
	return true
}

// Adds specific properties to the Cluster Initialized event.
// The event is triggered when a previously posted cluster changes state from
// UNINITIALIZED to something else.
func clusterInitialized(rp *phonehome.RequestParams, props map[string]any) bool {
	if rp.Path != "/v1.ClustersService/PutCluster" {
		return false
	}
	if req, ok := rp.GRPCReq.(*storage.Cluster); ok {
		mux.Lock()
		defer mux.Unlock()

		newStatus := req.GetHealthStatus().GetSensorHealthStatus()
		if newStatus == storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Add(req.GetId())
		} else
		// Fire an event if the sensor moves from UNINITIALIZED state.
		// The event will be missed if the central restarts between
		// postCluster and first putCluster.
		if uninitializedClusters.Contains(req.GetId()) &&
			newStatus != storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Remove(req.GetId())
			props["Code"] = rp.Code
			props["Cluster Type"] = req.GetType().String()
			props["Cluster ID"] = req.GetId()
			props["Managed By"] = req.GetManagedBy().String()
			return true
		}
	}
	return false
}

// Adds properties to the roxctl event.
func roxctl(rp *phonehome.RequestParams, props map[string]any) bool {
	if !strings.Contains(rp.UserAgent, "roxctl") {
		return false
	}
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["User-Agent"] = rp.UserAgent
	props["Method"] = rp.GetMethod()
	props["Protocol"] = rp.GetProtocol()
	return true
}
