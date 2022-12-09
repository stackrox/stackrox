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

	clusterStatus = map[string]storage.ClusterHealthStatus_HealthStatusLabel{}
	mux           = &sync.Mutex{}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call":            {apiCall},
		"Cluster Registered":  {postCluster},
		"Cluster Initialized": {putCluster},
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
		return true
	}
	return false
}

// Adds Post Cluster call specific properties to the Post Cluster event.
func postCluster(rp *phonehome.RequestParams, props map[string]any) bool {
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
		clusterStatus[req.GetId()] = req.GetHealthStatus().GetSensorHealthStatus()
	}
	return true
}

// Adds Put Cluster call specific properties to the Put Cluster event.
// The event is triggered when a previously posted cluster changes state from
// UNINITIALIZED to something else.
func putCluster(rp *phonehome.RequestParams, props map[string]any) bool {
	if rp.Path != "/v1.ClustersService/PutCluster" {
		return false
	}
	if req, ok := rp.GRPCReq.(*storage.Cluster); ok {
		mux.Lock()
		defer mux.Unlock()
		lastStatus, ok := clusterStatus[req.GetId()]
		newStatus := req.GetHealthStatus().GetSensorHealthStatus()
		if !ok && newStatus == storage.ClusterHealthStatus_UNINITIALIZED {
			clusterStatus[req.GetId()] = newStatus
		} else if lastStatus == storage.ClusterHealthStatus_UNINITIALIZED &&
			newStatus != lastStatus {
			delete(clusterStatus, req.GetId())
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
	if rp.HTTPReq != nil {
		props["Protocol"] = "HTTP"
	} else {
		props["Protocol"] = "GRPC"
	}
	return true
}
