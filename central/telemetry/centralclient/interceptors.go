package centralclient

import (
	"net/http"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

var (
	trackedPaths []string
	ignoredPaths = []string{"/v1/ping", "/v1.PingService/Ping", "/v1/metadata", "/static/*"}

	uninitializedClusters     = set.NewSet[string]()
	uninitializedClustersLock = &sync.Mutex{}

	interceptors = map[string][]phonehome.Interceptor{
		"API Call":            {apiCall},
		"Cluster Registered":  {clusterRegistered},
		"Cluster Initialized": {clusterInitialized},
		"roxctl":              {roxctl},
	}
)

// apiCall enables API Call events for the API paths specified in the
// trackedPaths ("*" value enables all paths) and have no match in the
// ignoredPaths list.
func apiCall(rp *phonehome.RequestParams, props map[string]any) bool {
	if !rp.HasPathIn(ignoredPaths) && rp.HasPathIn(trackedPaths) {
		props["Path"] = rp.Path
		props["Code"] = rp.Code
		props["User-Agent"] = rp.UserAgent
		props["Method"] = rp.Method
		return true
	}
	return false
}

var postCluster = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.ClustersService/PostCluster",
	HTTPMethod: http.MethodPost,
	HTTPPath:   "/v1/cluster",
}

// clusterRegistered enables the Cluster Registered event and adds specific
// properties.
func clusterRegistered(rp *phonehome.RequestParams, props map[string]any) bool {
	if !rp.Is(postCluster) {
		return false
	}

	props["Code"] = rp.Code
	if cluster := phonehome.GetGRPCRequestBody(v1.ClustersServiceServer.PostCluster, rp); cluster != nil {
		props["Cluster Type"] = cluster.GetType().String()
		props["Cluster ID"] = cluster.GetId()
		props["Managed By"] = cluster.GetManagedBy().String()
		uninitializedClustersLock.Lock()
		defer uninitializedClustersLock.Unlock()
		if cluster.GetHealthStatus().GetSensorHealthStatus() == storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Add(cluster.GetId())
		}
	}
	return true
}

var putCluster = &phonehome.ServiceMethod{
	GRPCMethod: "/v1.ClustersService/PutCluster",
	HTTPMethod: http.MethodPut,
	HTTPPath:   "/v1/cluster/*",
}

// clusterInitialized enables the Cluster Initialized event and adds specific
// properties.
func clusterInitialized(rp *phonehome.RequestParams, props map[string]any) bool {
	if !rp.Is(putCluster) {
		return false
	}

	if cluster := phonehome.GetGRPCRequestBody(v1.ClustersServiceServer.PutCluster, rp); cluster != nil {
		uninitializedClustersLock.Lock()
		defer uninitializedClustersLock.Unlock()

		newStatus := cluster.GetHealthStatus().GetSensorHealthStatus()
		if newStatus == storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Add(cluster.GetId())
		} else
		// Fire an event if the sensor moves from UNINITIALIZED state.
		// The event will be missed if the central restarts between
		// postCluster and first putCluster.
		if uninitializedClusters.Contains(cluster.GetId()) &&
			newStatus != storage.ClusterHealthStatus_UNINITIALIZED {
			uninitializedClusters.Remove(cluster.GetId())
			props["Code"] = rp.Code
			props["Cluster Type"] = cluster.GetType().String()
			props["Cluster ID"] = cluster.GetId()
			props["Managed By"] = cluster.GetManagedBy().String()
			return true
		}
	}
	return false
}

// roxctl enables the roxctl event.
func roxctl(rp *phonehome.RequestParams, props map[string]any) bool {
	if !strings.Contains(rp.UserAgent, "roxctl") {
		return false
	}
	props["Path"] = rp.Path
	props["Code"] = rp.Code
	props["User-Agent"] = rp.UserAgent
	props["Method"] = rp.Method
	return true
}
